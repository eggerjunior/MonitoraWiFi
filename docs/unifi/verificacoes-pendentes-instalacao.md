# Verificações Pendentes na Instalação UniFi Real

> **Escopo confirmado em 2026-08-01**: apenas o console **Cloud Gateway Max
> (UCG Max)** da Egger está em escopo deste projeto. O usuário tem acesso a
> um segundo console ("Dream Router 7") vinculado à mesma conta Ubiquiti,
> mas esse pertence à casa do filho dele — **fora de escopo**, não deve ser
> tratado/inventariado por este produto.

Estes itens **não podem ser confirmados a partir de documentação pública** — exigem
acesso direto ao console (Cloud Gateway Max) da instalação Egger. Nenhum deles deve
ser assumido/simulado antes de resposta real; a Fase 3 (integração UniFi) trata cada
"a validar" da capability matrix como bloqueado até esta lista ser respondida.

## Versões e disponibilidade de API

1. ✅ **Confirmado em 2026-08-01** (print de `Settings → UCG Max → Updates`):
   **UniFi OS 5.1.19** (UCG Max, "Up to Date", canal Official).
2. ✅ **Confirmado em 2026-08-01**: **UniFi Network 10.5.67** ("Up to Date",
   canal Official). Outras aplicações no mesmo console: Protect 7.1.87,
   InnerSpace 1.3.21 (Access/Talk/Connect não instaladas).
3. ✅ **Confirmado em 2026-08-01**: a Network API local (UniFi Network
   10.5.67) exige gerar uma API key manualmente — tela `Integrations` do
   próprio console (Network app), não vem habilitada "por padrão" no
   sentido de dispensar essa etapa. O console já tinha 3 keys anteriores
   (Home Assistant x2, uma sem nome claro) — o usuário criou uma nova,
   nomeada "MonitoraWiFi". **A key em si nunca foi nem será registrada
   neste repositório** (segredo puro, gerado uma única vez pela UI da
   Ubiquiti) — guardada só no gerenciador de senhas do usuário.
4. ✅ **Confirmado em 2026-08-01**: método de autenticação é API key via
   header `X-API-KEY` (não usuário/senha) — confirmado pelo próprio exemplo
   de `curl` que a tela de Integrations mostra:
   `curl -k -X GET 'https://<ip-do-console>/proxy/network/integration/v1/sites' -H 'X-API-KEY: ...' -H 'Accept: application/json'`.
   **Achado extra**: o IP de gerenciamento do console é `192.168.110.1`,
   batendo com a sub-rede `Egger_Principal 192.168.110.0/24` esperada
   (ver item 11) — primeira confirmação real desse dado.
5. **Parcialmente confirmado em 2026-08-01**: existe conta Ubiquiti vinculada
   (login ativo em `unifi.ui.com`, visível no print). Usuário ainda não
   decidiu se quer habilitar a Site Manager API (cloud, opcional) ou operar
   100% self-hosted — decisão adiada, não bloqueia o restante do
   levantamento.

## Campos realmente expostos pela Network API local nesta versão

6. ✅ **Confirmado em 2026-08-02** (via `GET .../devices/{id}` real contra um
   U7 Pro): só um **subconjunto**. Por rádio, a API expõe apenas
   `wlanStandard`, `frequencyGHz`, `channelWidthMHz` e `channel` — os três
   rádios (2.4/5/6GHz) desta instalação reportam `wlanStandard: "802.11be"`
   (Wi-Fi 7), banda 6GHz em canal 117/80MHz. **Não expostos nesta versão**:
   potência de transmissão, utilização do canal, número de clientes por
   rádio, airtime, retries, PHY rate — nenhum desses campos aparece na
   resposta. Capability matrix deve marcar canal/largura como
   "confirmado", o resto como "indisponível nesta API/versão", não "a
   validar".
7. ✅ **Confirmado em 2026-08-02** (via `GET .../devices/{id}` real contra
   a USW Lite 16 PoE): também só um **subconjunto**. Por porta, a API
   expõe `idx`, `state` (UP/DOWN), `connector`, `maxSpeedMbps`,
   `speedMbps` (velocidade negociada) e `poe.{standard,type,enabled,state}`.
   **Não expostos nesta versão**: contadores RX/TX, erros, CRC, flaps,
   consumo PoE em watts, orçamento PoE total. Mesma conclusão do item 6 —
   capability matrix marca esses campos como "indisponível", não "a
   validar".
8. ✅ **Confirmado em 2026-08-02**: **não existe endpoint de
   eventos/alarmes na Network API local integration v1** desta versão —
   `GET .../alarms` e `GET .../events` retornam 404 explícito
   (`"No endpoint GET /integration/v1/sites/{id}/alarms"`). Eventos/alarmes
   **não podem ser entregues via polling desta API** — as únicas rotas que
   sobram são syslog (item 18: **não configurado** nesta instalação) ou a
   Site Manager API (cloud, item 5: ainda não decidido se será habilitada).
   Sem uma dessas duas, a Fase 3 "eventos/alarmes" fica bloqueada
   estruturalmente nesta instalação, não é falta de implementação.
9. ✅ **Confirmado e implementado em 2026-08-02**: topologia **dispositivo
   → dispositivo** vem pronta em `GET .../devices/{id}` — campo
   `uplink.deviceId` aponta pro dispositivo upstream (o AP testado aponta
   pro switch; o switch testado aponta pro Cloud Gateway Max). Não
   precisa inferir por LLDP, igual ao que já valia pra topologia
   cliente→dispositivo (`uplinkDeviceId` em `/clients`). **Em produção**:
   agente busca o detalhe de cada device pra extrair o uplink, backend
   persiste (`unifi_devices.uplink_device_id`, migração 0014), web mostra
   árvore gateway→switch→AP na tela Dispositivos, iOS mostra "Conectado
   a" por dispositivo.
10. ✅ **Confirmado em 2026-08-02** (via `GET .../clients` real, 5 clientes
    de amostra): **nenhum campo de DPI/categoria/aplicação** aparece na
    resposta — o objeto de cliente só tem `type`, `id`, `name`,
    `connectedAt`, `ipAddress`, `macAddress`, `uplinkDeviceId`,
    `access.type`. DPI não é exposto por este endpoint nesta versão (não
    dá pra distinguir se está desabilitado no console ou se é só a API que
    não expõe — mas pro propósito deste levantamento, o resultado prático
    é o mesmo: não há dado de DPI disponível pra consumir agora).

## Configuração de rede da instalação (para validar contra o modelo de dados)

11. ✅ **Parcialmente confirmado em 2026-08-01** (via `GET .../sites/{id}/clients`,
    primeira página de 80 clientes): `192.168.110.0/24` (Principal) e
    `192.168.120.0/24` (IoT) confirmadas com clientes reais conectados
    agora. `192.168.130.0/24` (Convidados) ainda não observada nessa
    amostra — não significa que não exista, só que não apareceu na
    primeira página/nenhum cliente conectado nela no momento da consulta.
    **Achado relevante**: o inventário real de dispositivos (14 ao todo) é
    **maior que o documento-fonte original presumia** ("4 APs U7 Pro + 1
    switch") — há também múltiplos switches adicionais e outros
    APs/repetidores de um modelo não mencionado no documento-fonte. Contagem
    exata e modelos registrados em `docs/architecture/05-modelo-dados.md`
    quando o levantamento for consolidado; nomes/MACs/localização exatos de
    dispositivos e clientes **não são registrados em nenhum documento deste
    repositório público** (o inventário revela o layout físico da
    residência — câmeras, cômodos — que não deve ficar em um repo público).
12. ✅ **Confirmado em 2026-08-02**: IPv6 **não está habilitado** em
    nenhuma das redes desta instalação.
13. ✅ **Confirmado em 2026-08-02**: **não há VPN configurada** no
    gateway.
14. ✅ **Confirmado em 2026-08-02**: **não há regras de firewall/IDS-IPS**
    habilitadas além do padrão do UniFi.

## Hardware e licenciamento

15. ✅ **Confirmado em 2026-08-02**: as duas WANs estão fisicamente
    conectadas e configuradas (failover/balanceamento) no Cloud Gateway
    Max — não é um cenário só planejado.
16. ✅ **Confirmado em 2026-08-01** (via `GET .../sites/{id}/devices`):
    U7 Pro (4 unidades) firmware **8.7.11**; USW Lite 16 PoE firmware
    **7.4.1**; Cloud Gateway Max firmware **5.1.19** (mesma versão do
    UniFi OS, esperado). Se MLO/Wi-Fi 7 já está habilitado na prática
    ainda não foi checado — depende de configuração de rádio, não só de
    firmware (ver item 6).
17. ✅ **Confirmado em 2026-08-02** (print real de `Settings → [seção de
    Registro de Tráfego] → Monitoramento SNMP`): **SNMP não está
    habilitado** — nem "Versão 1/2C" nem "Versão 3" marcados.
18. ✅ **Confirmado em 2026-08-02**: **syslog não está configurado** para
    nenhum coletor hoje.

## Processo de validação recomendado

Cada resposta obtida deve:
1. Atualizar a linha correspondente em `docs/unifi/capability-matrix.md` de
   "a validar" para "confirmado" ou "indisponível", com a data da verificação.
2. Quando a resposta implicar mudança no modelo de dados
   (`docs/architecture/05-modelo-dados.md`), registrar isso como ajuste antes de
   escrever a migração definitiva da Fase 3.
3. Quando a resposta revelar uma limitação not documentada aqui, adicionar uma nova
   linha neste arquivo e no threat model, se aplicável.
