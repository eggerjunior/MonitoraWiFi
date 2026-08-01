#!/usr/bin/env python3
"""Cria o Bundle ID no App Store Connect via API e verifica se o app record
já existe. O app record em si NÃO pode ser criado via API (POST /v1/apps
retorna 403 FORBIDDEN_ERROR para chaves de API — restrição permanente da
Apple, não um erro pontual desta chave). Essa etapa fica sempre com o
Ildemar: criar manualmente em App Store Connect > Apps > "+" > New App,
usando o mesmo Bundle ID (já aparece na lista), SKU = bundle id,
Primary Language pt-BR.

Uso:
    source scripts/asc.env  # ou defina ASC_KEY_ID/ASC_ISSUER_ID/ASC_KEY_PATH no ambiente
    python3 scripts/create_app.py
"""
import json
import os
import sys
import time
import urllib.request
import urllib.error

import jwt

BUNDLE_ID = "br.app.egger.network-intelligence"
APP_NAME = "Egger Network Intelligence"
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
    payload = {
        "iss": issuer_id,
        "iat": now,
        "exp": now + 600,
        "aud": "appstoreconnect-v1",
    }
    return jwt.encode(payload, private_key, algorithm="ES256", headers={"kid": key_id, "typ": "JWT"})


def api_request(token, method, path, body=None):
    url = f"{API_BASE}{path}"
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Authorization", f"Bearer {token}")
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req) as resp:
            return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read().decode() or "{}")


def main():
    key_id, issuer_id, private_key = load_config()
    token = make_jwt(key_id, issuer_id, private_key)

    print(f"==> Verificando Bundle ID {BUNDLE_ID}...")
    status, payload = api_request(token, "GET", f"/bundleIds?filter[identifier]={BUNDLE_ID}")
    if status != 200:
        sys.exit(f"ERRO ao consultar bundle id: {status} {payload}")

    bundle_id_resource_id = None
    if payload.get("data"):
        bundle_id_resource_id = payload["data"][0]["id"]
        print(f"    Já existe (id={bundle_id_resource_id}).")
    else:
        print("    Não existe — criando via POST /v1/bundleIds...")
        body = {
            "data": {
                "type": "bundleIds",
                "attributes": {
                    "identifier": BUNDLE_ID,
                    "name": APP_NAME,
                    "platform": "IOS",
                },
            }
        }
        status, payload = api_request(token, "POST", "/bundleIds", body)
        if status not in (200, 201):
            sys.exit(f"ERRO ao criar bundle id: {status} {payload}")
        bundle_id_resource_id = payload["data"]["id"]
        print(f"    Criado com sucesso (id={bundle_id_resource_id}).")

    print(f"==> Verificando app record para o Bundle ID {BUNDLE_ID}...")
    status, payload = api_request(token, "GET", f"/apps?filter[bundleId]={BUNDLE_ID}")
    if status != 200:
        sys.exit(f"ERRO ao consultar app record: {status} {payload}")

    if payload.get("data"):
        app_id = payload["data"][0]["id"]
        print(f"    App record já existe (id={app_id}). Fluxo de archive/TestFlight pode prosseguir.")
    else:
        print("    App record NÃO existe ainda.")
        print("    Isso é esperado — a Apple não permite criar app record via API")
        print("    (POST /v1/apps retorna 403 FORBIDDEN_ERROR para qualquer chave).")
        print("    PENDENTE (ação manual do Ildemar):")
        print(f"      1. App Store Connect > Apps > \"+\" > New App")
        print(f"      2. Bundle ID: {BUNDLE_ID} (já vai aparecer na lista)")
        print(f"      3. SKU: {BUNDLE_ID}")
        print(f"      4. Primary Language: pt-BR")
        print(f"      5. Avisar quando terminar (~1 minuto)")
        print(f"    Depois disso, rodar este script de novo (ou deixar o CI rodar) — ele vai")
        print(f"    encontrar o app record e o fluxo de TestFlight segue sem mais intervenção.")


if __name__ == "__main__":
    main()
