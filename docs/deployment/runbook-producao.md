# Runbook de produção

Primeiro runbook formal do projeto (Fase 8, 2026-08-02) — passos reais,
confirmados nesta e em sessões anteriores (ver
`docs/development-handoff/RELEASE_LOG.md` pra histórico completo de cada
deploy). Ambiente: VPS única (`wifi.egger.app.br`), acesso via SSH com a
chave `monitoravps_deploy`, checkout do repositório em
`/opt/projetos/MonitoraWiFi` no próprio servidor (as imagens são
construídas a partir desse checkout, não copiadas de fora).

## Topologia

Três containers Docker na rede `monitorawifi_net`:

- `monitorawifi-postgres` (`postgres:16-alpine`, volume
  `/opt/data/monitorawifi/postgres`, sem porta publicada)
- `monitorawifi-api` (sem porta publicada, só acessível pela rede interna)
- `monitorawifi-web` (publicado em `127.0.0.1:8421`, exposto ao público via
  nginx do Hestia em `wifi.egger.app.br`)

Configuração compartilhada (senha de banco, `DATABASE_URL`) em
`/opt/apps/monitorawifi/.env` — **nunca commitado**. O container `web`
precisa também de `API_BASE_URL=http://monitorawifi-api:8080/api/v1`
passado explicitamente (não vem do `.env` compartilhado — achado real de
2026-08-02, ver RELEASE_LOG: um redeploy usando só `--env-file` deixou
essa variável de fora e o web ficou tentando falar com `localhost:8080`).

Worker de anomalias (Fase 7) roda via `docker run --rm` avulso, agendado
por cron do host a cada 6h (não é um container de longa duração).

## Deploy de código novo (api e/ou web)

1. **Backup antes de qualquer migração**:
   ```bash
   ssh -i /caminho/monitoravps_deploy root@<host> \
     "bash /opt/projetos/MonitoraWiFi/infrastructure/scripts/backup-postgres.sh"
   ```
2. **Atualizar o checkout** (o código já commitado/pushado pro GitHub):
   ```bash
   ssh ... "cd /opt/projetos/MonitoraWiFi && git pull origin main"
   ```
3. **Buildar as imagens** (tag = hash curto do commit, `git rev-parse
   --short=8 HEAD`):
   ```bash
   ssh ... "cd /opt/projetos/MonitoraWiFi && \
     docker build -f apps/api/Dockerfile --build-arg GIT_COMMIT=<hash> -t monitorawifi-api:<hash> . && \
     docker build -f apps/web/Dockerfile --build-arg GIT_COMMIT=<hash> --build-arg BUILD_DATE=\$(date -u +%Y-%m-%dT%H:%M:%SZ) -t monitorawifi-web:<hash> ."
   ```
4. **Aplicar migrações pendentes** (se houver — o binário `migrate` já
   vem dentro da imagem da API, `/app/migrate` + `/app/migrations`):
   ```bash
   # confirmar versão atual primeiro
   ssh ... 'docker run --rm --network monitorawifi_net --env-file /opt/apps/monitorawifi/.env \
     --entrypoint /app/migrate monitorawifi-api:<hash> -path /app/migrations version'
   # aplicar
   ssh ... 'docker run --rm --network monitorawifi_net --env-file /opt/apps/monitorawifi/.env \
     --entrypoint /app/migrate monitorawifi-api:<hash> -path /app/migrations up'
   ```
   **Nunca** passar `-database "$DATABASE_URL"` explicitamente num
   comando SSH remoto — a variável não existe no shell local, só dentro
   do container via `--env-file`; deixar o flag de fora faz o binário
   usar `os.Getenv("DATABASE_URL")` corretamente.
5. **Substituir os containers**:
   ```bash
   ssh ... '
   docker stop monitorawifi-api monitorawifi-web
   docker rm monitorawifi-api monitorawifi-web
   docker run -d --name monitorawifi-api --network monitorawifi_net \
     --env-file /opt/apps/monitorawifi/.env --restart unless-stopped monitorawifi-api:<hash>
   docker run -d --name monitorawifi-web --network monitorawifi_net \
     -e API_BASE_URL=http://monitorawifi-api:8080/api/v1 \
     -p 127.0.0.1:8421:3000 --restart unless-stopped monitorawifi-web:<hash>
   '
   ```
6. **Verificar saúde** (o container distroless da API não tem
   shell/curl — usar um container efêmero na mesma rede):
   ```bash
   ssh ... 'docker run --rm --network monitorawifi_net curlimages/curl:8.10.1 \
     -s -o /dev/null -w "healthz=%{http_code}\n" http://monitorawifi-api:8080/healthz'
   curl -s -o /dev/null -w "%{http_code}\n" https://wifi.egger.app.br/login
   ```
7. Registrar o deploy em `docs/development-handoff/RELEASE_LOG.md`
   (commit, migrações aplicadas, resultado da verificação de saúde).

### Rollback

Sem migração nova envolvida: parar/remover o container e subir de novo
com a tag da imagem anterior (`docker images | grep monitorawifi-api`
lista todas as versões já construídas — nunca removidas automaticamente).
Com migração nova envolvida: rodar `migrate ... down 1` (reverte uma
migração) antes de voltar pra imagem anterior, ou restaurar o backup do
passo 1 se a migração já tiver corrompido dado.

## Backup e restore do Postgres

- **Backup**: `infrastructure/scripts/backup-postgres.sh` — roda via cron
  diário do host (não dentro de container, sobrevive a um `docker rm` do
  Postgres). Retenção: 14 dias. Destino:
  `/opt/data/monitorawifi/backups/backup-<timestamp>.sql.gz`.
- **Restore** (testado ponta a ponta na Fase 8):
  ```bash
  gunzip -c /opt/data/monitorawifi/backups/backup-<timestamp>.sql.gz | \
    docker exec -i monitorawifi-postgres psql -U monitorawifi -d monitorawifi
  ```
  Confirmar depois com uma query simples (`SELECT count(*) FROM
  agent_commands;`) que o dado esperado voltou.

## Enrolamento de um agente novo

1. Criar um token de uso único: `POST
   /api/v1/sites/{siteId}/agent-enrollment-tokens` (autenticado, permissão
   `manage_integrations`).
2. No host onde o agente vai rodar (**precisa estar na mesma LAN do
   console UniFi** — não numa VPS externa, senão não alcança
   `192.168.x.1`):
   ```bash
   docker build -t egger-agent https://github.com/eggerjunior/MonitoraWiFi.git#main -f apps/local-agent/Dockerfile
   docker run -d --name egger-agent --restart unless-stopped \
     -e BACKEND_URL=https://wifi.egger.app.br/api/v1 \
     -e ENROLLMENT_TOKEN=<token do passo 1> \
     -e UNIFI_BASE_URL=https://192.168.x.1 \
     -e UNIFI_API_KEY=<key gerada no console UniFi> \
     -e UNIFI_SITE_ID=<site id do console UniFi> \
     -v /caminho/persistente:/data \
     egger-agent
   ```
3. Confirmar heartbeat: `GET /api/v1/sites/{siteId}/agents` deve mostrar
   `last_seen_at` atualizando a cada poucos segundos.
4. Revogar um agente comprometido/descomissionado: não existe endpoint
   de revogação exposto ainda — hoje é feito via `UPDATE agents SET
   revoked_at = now()` direto no Postgres (pendência real: expor isso
   como ação de API, ver Fase 8 "faltam").
