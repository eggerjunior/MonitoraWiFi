#!/usr/bin/env python3
"""Cria um certificado de distribuição (Apple Distribution) + perfil de
provisionamento App Store para o app, reaproveitáveis em todo run do CI —
evita o problema documentado em references/ildemar-ios-release.md
("Certificados de assinatura esgotados"): Automatic signing cria um
certificado novo a cada archive, estourando a cota da conta Apple depois de
poucas execuções.

A chave privada e o .p12 nunca tocam disco fora de um diretório temporário
apagado ao final, e os valores gerados **nunca são impressos** — este script
os envia direto para `gh secret set` no repositório informado. Se `gh` não
estiver disponível/autenticado, ele falha em vez de imprimir os segredos
como alternativa.

Uso:
    source scripts/asc.env  # ou defina ASC_KEY_ID/ASC_ISSUER_ID/ASC_KEY_PATH
    python3 -m pip install --break-system-packages PyJWT cryptography
    python3 scripts/create_dist_cert.py eggerjunior/MonitoraWiFi
"""
import base64
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

import jwt
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.x509.oid import NameOID

BUNDLE_ID = "br.app.egger.network-intelligence"
PROFILE_NAME = "EggerNetworkIntelligence App Store"
API_BASE = "https://api.appstoreconnect.apple.com/v1"


def load_config():
    key_id = os.environ.get("ASC_KEY_ID")
    issuer_id = os.environ.get("ASC_ISSUER_ID")
    key_path = os.environ.get("ASC_KEY_PATH")
    if not (key_id and issuer_id and key_path):
        sys.exit("ERRO: defina ASC_KEY_ID, ASC_ISSUER_ID e ASC_KEY_PATH (ver scripts/asc.env.example)")
    if not os.path.isfile(key_path):
        sys.exit(f"ERRO: chave .p8 não encontrada em: {key_path}")
    with open(key_path, "r") as f:
        private_key = f.read()
    return key_id, issuer_id, private_key


def make_jwt(key_id, issuer_id, private_key):
    now = int(time.time())
    payload = {"iss": issuer_id, "iat": now, "exp": now + 1200, "aud": "appstoreconnect-v1"}
    return jwt.encode(payload, private_key, algorithm="ES256", headers={"kid": key_id, "typ": "JWT"})


def api_request(token, method, path, body=None):
    url = f"{API_BASE}{path}"
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Authorization", f"Bearer {token}")
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req) as resp:
            raw = resp.read()
            return resp.status, (json.loads(raw.decode()) if raw else {})
    except urllib.error.HTTPError as e:
        raw = e.read()
        return e.code, (json.loads(raw.decode()) if raw else {})


def gh_secret_set(repo: str, name: str, value: str):
    if shutil.which("gh") is None:
        sys.exit(f"ERRO: 'gh' não encontrado — não é seguro imprimir {name}. Instale/autentique o gh CLI.")
    result = subprocess.run(
        ["gh", "secret", "set", name, "--repo", repo],
        input=value.encode(),
        capture_output=True,
    )
    if result.returncode != 0:
        sys.exit(f"ERRO ao definir secret {name}: {result.stderr.decode()}")
    print(f"    Secret {name} configurado ({len(value)} bytes).")


def main():
    if len(sys.argv) != 2:
        sys.exit("uso: create_dist_cert.py <owner/repo>")
    repo = sys.argv[1]

    key_id, issuer_id, private_key_pem = load_config()
    token = make_jwt(key_id, issuer_id, private_key_pem)

    print("==> Buscando Bundle ID...")
    status, payload = api_request(token, "GET", f"/bundleIds?filter[identifier]={BUNDLE_ID}")
    if status != 200 or not payload.get("data"):
        sys.exit(f"ERRO: bundle id {BUNDLE_ID} não encontrado ({status}): {payload}. Rode create_app.py primeiro.")
    bundle_id_resource_id = payload["data"][0]["id"]
    print(f"    id={bundle_id_resource_id}")

    with tempfile.TemporaryDirectory() as tmpdir:
        key_path = os.path.join(tmpdir, "dist.key.pem")
        cert_path = os.path.join(tmpdir, "dist.cert.pem")
        p12_path = os.path.join(tmpdir, "dist.p12")

        print("==> Gerando chave privada RSA 2048 + CSR (só em disco temporário local, apagado ao final)...")
        key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
        csr = (
            x509.CertificateSigningRequestBuilder()
            .subject_name(x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "Egger Network Intelligence Distribution")]))
            .sign(key, hashes.SHA256())
        )
        csr_der_b64 = base64.b64encode(csr.public_bytes(serialization.Encoding.DER)).decode()

        with open(key_path, "wb") as f:
            f.write(key.private_bytes(
                encoding=serialization.Encoding.PEM,
                format=serialization.PrivateFormat.TraditionalOpenSSL,
                encryption_algorithm=serialization.NoEncryption(),
            ))

        print("==> Solicitando certificado de distribuição via POST /v1/certificates...")
        body = {
            "data": {
                "type": "certificates",
                "attributes": {"certificateType": "IOS_DISTRIBUTION", "csrContent": csr_der_b64},
            }
        }
        status, payload = api_request(token, "POST", "/certificates", body)
        if status not in (200, 201):
            sys.exit(f"ERRO ao criar certificado ({status}): {payload}")
        cert_id = payload["data"]["id"]
        cert_content_b64 = payload["data"]["attributes"]["certificateContent"]
        print(f"    Certificado criado (id={cert_id}).")

        with open(cert_path, "wb") as f:
            f.write(x509.load_der_x509_certificate(base64.b64decode(cert_content_b64)).public_bytes(serialization.Encoding.PEM))

        print("==> Empacotando .p12 (openssl -legacy, exigido pelo 'security import' do macOS)...")
        p12_password = base64.urlsafe_b64encode(os.urandom(18)).decode().rstrip("=")
        subprocess.run(
            [
                "openssl", "pkcs12", "-export", "-legacy",
                "-inkey", key_path, "-in", cert_path,
                "-out", p12_path, "-passout", f"pass:{p12_password}",
            ],
            check=True,
        )
        with open(p12_path, "rb") as f:
            p12_b64 = base64.b64encode(f.read()).decode()

        print("==> Verificando/recriando perfil de provisionamento App Store...")
        status, payload = api_request(token, "GET", f"/profiles?filter[name]={urllib.parse.quote(PROFILE_NAME)}")
        if status == 200:
            for existing in payload.get("data", []):
                print(f"    Removendo perfil existente (id={existing['id']})...")
                api_request(token, "DELETE", f"/profiles/{existing['id']}")

        profile_body = {
            "data": {
                "type": "profiles",
                "attributes": {"name": PROFILE_NAME, "profileType": "IOS_APP_STORE"},
                "relationships": {
                    "bundleId": {"data": {"type": "bundleIds", "id": bundle_id_resource_id}},
                    "certificates": {"data": [{"type": "certificates", "id": cert_id}]},
                },
            }
        }
        status, payload = api_request(token, "POST", "/profiles", profile_body)
        if status not in (200, 201):
            sys.exit(f"ERRO ao criar perfil ({status}): {payload}")
        profile_content_b64 = payload["data"]["attributes"]["profileContent"]
        print(f"    Perfil criado (id={payload['data']['id']}).")

        print("==> Configurando secrets no repositório (valores nunca impressos)...")
        gh_secret_set(repo, "IOS_DIST_CERT_P12_BASE64", p12_b64)
        gh_secret_set(repo, "IOS_DIST_CERT_PASSWORD", p12_password)
        gh_secret_set(repo, "IOS_DIST_PROFILE_BASE64", profile_content_b64)

    print("")
    print("==> OK: certificado/perfil de distribuição criados e secrets configurados.")


if __name__ == "__main__":
    main()
