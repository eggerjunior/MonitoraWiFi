# infrastructure/scripts — backup e restore (Fase 8)

## Backup

`backup-postgres.sh` formaliza o processo manual usado durante o
desenvolvimento (um `pg_dump` antes de cada migração, ver
`docs/development-handoff/RELEASE_LOG.md`) — agora com compressão e
retenção automática.

```bash
# Variáveis com default sensato para o host de produção atual (Hestia):
POSTGRES_CONTAINER=monitorawifi-postgres   # nome do container
ENV_FILE=/opt/apps/monitorawifi/.env       # onde estão POSTGRES_USER/DB
BACKUP_DIR=/opt/data/monitorawifi/backups  # destino dos .sql.gz
RETENTION_DAYS=14                          # backups mais antigos são apagados

./backup-postgres.sh
```

Testado ponta a ponta com um Postgres efêmero real (não simulado):
backup gerado, validado como dump real (`zcat | head`), e restaurado com
sucesso em um banco novo.

### Agendamento (cron do host)

Recomendado: diário, fora do horário de pico. Exemplo de crontab
(`crontab -e` como root no host):

```cron
0 3 * * * BACKUP_DIR=/opt/data/monitorawifi/backups /opt/projetos/MonitoraWiFi/infrastructure/scripts/backup-postgres.sh >> /var/log/monitorawifi-backup.log 2>&1
```

## Restore

```bash
gunzip -c /opt/data/monitorawifi/backups/backup-<timestamp>.sql.gz \
  | docker exec -i monitorawifi-postgres psql -U <POSTGRES_USER> -d <POSTGRES_DB>
```

**Atenção**: isso aplica o dump sobre o banco existente (não recria do
zero) — se a intenção for restaurar para um estado limpo, apagar/recriar
o banco antes, ou restaurar num container novo primeiro para validar.
