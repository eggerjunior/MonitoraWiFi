#!/usr/bin/env sh
# Instalação do agente local (Seção 22: "instalação simples: curl ... | sh").
#
# Uso:
#   curl -fsSL https://raw.githubusercontent.com/eggerjunior/MonitoraWiFi/main/apps/local-agent/scripts/install.sh \
#     | BACKEND_URL=https://api.egger-network-intelligence.example/api/v1 \
#       ENROLLMENT_TOKEN=<token gerado no painel> sh
#
# Requer root (cria usuário de sistema, unit do systemd, diretórios em /etc
# e /var/lib). Não abre nenhuma porta de entrada (ADR-001) — só instala o
# binário e a configuração de conexão outbound.
set -eu

REPO="https://github.com/eggerjunior/MonitoraWiFi"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/egger-agent"
DATA_DIR="/var/lib/egger-agent"

: "${BACKEND_URL:?defina BACKEND_URL (ex.: https://api.seu-dominio/api/v1)}"
: "${ENROLLMENT_TOKEN:?defina ENROLLMENT_TOKEN (gerado no painel para o site)}"

if [ "$(id -u)" -ne 0 ]; then
  echo "ERRO: rode como root (sudo)." >&2
  exit 1
fi

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "ERRO: arquitetura não suportada: $arch" >&2; exit 1 ;;
esac

asset="egger-agent-${os}-${arch}"
release_url="${REPO}/releases/latest/download/${asset}"

echo "==> Tentando baixar binário pré-compilado ($release_url)..."
if command -v curl >/dev/null 2>&1 && curl -fsSL -o "${INSTALL_DIR}/egger-agent" "$release_url" 2>/dev/null; then
  chmod +x "${INSTALL_DIR}/egger-agent"
  echo "==> Binário instalado em ${INSTALL_DIR}/egger-agent"
else
  echo "==> Nenhum release publicado ainda (ou download falhou)."
  echo "    Este projeto ainda não tem um pipeline de release do agente"
  echo "    (pendência registrada — ver apps/local-agent/README.md)."
  echo "    Alternativa: compilar localmente com 'go build' a partir do"
  echo "    checkout do repositório, ou usar o Dockerfile"
  echo "    (apps/local-agent/Dockerfile) em vez deste script."
  exit 1
fi

echo "==> Criando usuário de sistema egger-agent..."
if ! id egger-agent >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin egger-agent
fi

mkdir -p "$CONFIG_DIR" "$DATA_DIR"
chown egger-agent:egger-agent "$DATA_DIR"
chmod 700 "$DATA_DIR"

cat > "${CONFIG_DIR}/env" <<EOF
BACKEND_URL=${BACKEND_URL}
ENROLLMENT_TOKEN=${ENROLLMENT_TOKEN}
STATE_FILE=${CONFIG_DIR}/agent.json
QUEUE_FILE=${DATA_DIR}/queue.jsonl
EOF
chmod 600 "${CONFIG_DIR}/env"
chown egger-agent:egger-agent "${CONFIG_DIR}/env"

curl -fsSL -o /etc/systemd/system/egger-agent.service \
  "${REPO}/raw/main/apps/local-agent/scripts/egger-agent.service"

systemctl daemon-reload
systemctl enable --now egger-agent

echo ""
echo "==> Agente instalado e iniciado. Acompanhe com:"
echo "    journalctl -u egger-agent -f"
