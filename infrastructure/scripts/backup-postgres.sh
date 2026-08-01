#!/usr/bin/env bash
# Backup do Postgres de produção (Fase 8) — formaliza o que foi feito
# manualmente antes de cada migração durante o desenvolvimento (ver
# docs/development-handoff/RELEASE_LOG.md, entradas "pre-000X"). Pensado
# para rodar via cron do host (não dentro de um container, para sobreviver
# a um `docker rm` do container do Postgres).
#
# Uso: BACKUP_DIR=/opt/data/monitorawifi/backups ./backup-postgres.sh
#
# Retenção: mantém os últimos RETENTION_DAYS dias de backups diários,
# apaga o resto — nunca apaga o backup do dia corrente, mesmo que
# RETENTION_DAYS seja 0.
set -euo pipefail

CONTAINER="${POSTGRES_CONTAINER:-monitorawifi-postgres}"
ENV_FILE="${ENV_FILE:-/opt/apps/monitorawifi/.env}"
BACKUP_DIR="${BACKUP_DIR:-/opt/data/monitorawifi/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"

if [ ! -f "$ENV_FILE" ]; then
  echo "ERRO: arquivo de env não encontrado em $ENV_FILE" >&2
  exit 1
fi

# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a

if [ -z "${POSTGRES_USER:-}" ] || [ -z "${POSTGRES_DB:-}" ]; then
  echo "ERRO: POSTGRES_USER/POSTGRES_DB não definidos em $ENV_FILE" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"

timestamp="$(date +%Y%m%d-%H%M%S)"
dest="$BACKUP_DIR/backup-$timestamp.sql.gz"

echo "==> Gerando backup de $POSTGRES_DB em $dest"
docker exec "$CONTAINER" pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" | gzip > "$dest"

size="$(du -h "$dest" | cut -f1)"
echo "==> Backup concluído ($size)"

echo "==> Removendo backups com mais de $RETENTION_DAYS dias"
find "$BACKUP_DIR" -name "backup-*.sql.gz" -mtime "+$RETENTION_DAYS" -print -delete

echo "==> Backups atuais:"
ls -la "$BACKUP_DIR"
