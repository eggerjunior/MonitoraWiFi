# Threat Model Inicial — Egger Network Intelligence

Metodologia: STRIDE por componente, com foco nos vetores específicos deste sistema
(ferramentas de rede ativas, agente com presença na LAN, credenciais de integração
UniFi, dados de localização física via LiDAR).

## 1. Ativos a proteger

1. Credenciais/API keys da integração UniFi (acesso à Network API local e/ou Site
   Manager API).
2. Credenciais do agente local (identidade do agente perante o backend).
3. Dados de clientes de rede (MAC, IP, hostname, tráfego agregado) — dado pessoal sob
   LGPD quando associável a uma pessoa.
4. Malha 3D e plantas de imóveis capturadas via LiDAR — revela layout físico da
   residência (dado sensível de segurança física, não só de privacidade digital).
5. Sessões de usuário (web/iOS) e tokens de API.
6. Integridade dos resultados de diagnóstico (não podem ser falsificados/manipulados
   sem detecção, já que embasam decisões como "trocar AP" ou "abrir chamado com ISP").
7. Disponibilidade do próprio sistema de monitoramento (ironicamente, um sistema de
   observabilidade que cai é pior que não ter monitoramento, pois gera falsa
   sensação de segurança).

## 2. Superfícies de ataque por componente

### 2.1 Agente local (maior superfície nova introduzida pelo produto)

| Ameaça (STRIDE) | Cenário | Mitigação |
|---|---|---|
| Spoofing | Um agente falso se autentica como se fosse o agente legítimo do site | Credencial de agente única por instalação, rotacionável, armazenada fora do código; suporte a mTLS como alvo, credencial rotacionável como mínimo viável |
| Tampering | Comprometimento do host do agente é usado para forjar métricas (ex.: esconder um incidente real) | Backend valida limites de sanidade (rate, ranges plausíveis) e loga discrepâncias; auditoria de mudanças de configuração do agente |
| Repudiation | Ação disparada pelo agente (ex.: Wake-on-LAN) sem rastro de quem/quando pediu | Todo comando ao agente carrega `actor`, `timestamp`, `correlation_id`, registrado em AuditLog antes de ser enfileirado |
| Information Disclosure | Agente comprometido é usado como pivô para escanear/enumerar a LAN inteira do cliente | Port scan e descoberta só rodam em alvos/faixas explicitamente cadastrados pelo usuário (Seção 2.2); sem varredura "de toda a internet" a partir do agente |
| Denial of Service | Agente satura a LAN com testes ativos (ping/scan agressivo) | Rate limiting local no agente, limites de concorrência configuráveis, testes ativos com circuit breaker |
| Elevation of Privilege | Processo do agente roda com privilégios excessivos no host | Princípio do menor privilégio: agente não deve exigir root, exceto onde estritamente necessário (ex.: ICMP raw em alguns SOs) — documentar exceções explicitamente |

### 2.2 Backend central

| Ameaça | Cenário | Mitigação |
|---|---|---|
| Spoofing | Token de sessão web/iOS roubado | Sessões seguras (httpOnly/secure cookies ou tokens de curta duração + refresh), MFA, passkeys |
| Tampering | Requisição de API forjada para alterar configuração crítica (ex.: alterar canal remotamente) | Ações de alteração de rede desativadas por padrão (Seção 14); quando existirem, exigem permissão administrativa explícita + confirmação + diff visível |
| Repudiation | Admin nega ter executado uma ação destrutiva | AuditLog append-only, com actor, IP, timestamp, payload da mudança |
| Information Disclosure | Vazamento de dados de múltiplos clientes por falha de isolamento multi-tenant | Toda query filtrada por `organization_id`/`site_id` no nível de repositório, nunca deixado a cargo do handler; testes de isolamento entre tenants |
| Denial of Service | Abuso de endpoints de teste ativo (ex.: port scan) para transformar o backend em ferramenta de ataque a terceiros | Validação rigorosa de host/porta/CIDR (allowlist de RFC 1918 + hosts cadastrados), proteção anti-SSRF em qualquer endpoint que aceite URL/host do usuário |
| Elevation of Privilege | Escalação via RBAC mal implementado (ex.: Viewer conseguindo executar automações) | RBAC centralizado e testado por papel (Owner/Administrator/Operator/Viewer/Auditor), nunca checado só no frontend |

### 2.3 Integração UniFi

| Ameaça | Cenário | Mitigação |
|---|---|---|
| Spoofing | Console UniFi falso responde à integração para injetar dados forjados | TLS com validação de certificado (ou pinning quando aplicável) na conexão do agente ao console local |
| Tampering | Credencial da UniFi API vaza e é usada para reconfigurar a rede do cliente | Chave/credencial armazenada fora do código-fonte, escopo mínimo necessário (somente leitura quando a API permitir granularidade), rotação de chaves |
| Information Disclosure | Adaptador legado (não oficial) expõe mais dado do que o necessário e sem controle de versão | Adaptador legado isolado, desativado por padrão, habilitado apenas com consentimento explícito do administrador |

### 2.4 App iOS

| Ameaça | Cenário | Mitigação |
|---|---|---|
| Spoofing | App forjado consumindo a API com credenciais roubadas do Keychain | Keychain com proteção biométrica (LocalAuthentication) para operações sensíveis; certificate pinning avaliado no ADR de segurança de transporte |
| Information Disclosure | Malha 3D/planta da residência exposta por exportação insegura ou compartilhamento acidental | Exportação de relatórios de cobertura exige ação explícita; nunca sincronização automática para serviços de terceiros fora do backend do produto |
| Tampering | Man-in-the-middle em rede pública durante uso do app | TLS obrigatório ponta a ponta, HSTS no backend, sem fallback para HTTP |

### 2.5 Web

| Ameaça | Cenário | Mitigação |
|---|---|---|
| Cross-Site Scripting | Nome de dispositivo/cliente (vindo do UniFi, não confiável) renderizado sem escaping | Sanitização/escaping obrigatório de todo dado de origem externa (hostname, SSID, etc.) antes de renderizar |
| CSRF | Ação administrativa disparada por site malicioso usando sessão do usuário | Tokens CSRF ou SameSite cookies + verificação de origem em mutações |
| Session Hijacking | Roubo de sessão via XSS ou rede insegura | Cookies httpOnly/secure, rotação de sessão, expiração curta + refresh |

## 3. Riscos específicos deste domínio (além do STRIDE genérico)

1. **Ferramentas ativas como vetor de abuso de terceiros**: port scanner, HTTP client
   e traceroute do agente/app são, por natureza, ferramentas duais. Mitigação
   estrutural: todo alvo precisa estar cadastrado pelo usuário e dentro de escopos
   permitidos (RFC 1918 por padrão; alvo externo exige confirmação explícita e fica
   registrado em auditoria). Nenhuma ação de scan é acionável por automação sem
   aprovação humana prévia na configuração da regra.
2. **SSRF via HTTP client/webhooks**: qualquer campo que aceite URL/host arbitrário
   (HTTP client, webhook de alerta, integração) deve validar contra IPs internos da
   infraestrutura do próprio backend (não só contra a LAN do cliente) — um agente que
   testa "GET https://169.254.169.254/" ou endpoints do próprio cloud provider do
   backend é um vetor real de SSRF documentado na indústria.
3. **Dados de localização física (LiDAR)**: tratados como categoria própria de dado
   sensível, com retenção e exportação controladas separadamente de métricas de rede
   (ver Seção 19 do documento-fonte e ADR de privacidade).
4. **Command injection em qualquer wrapper de ferramenta de sistema**: se qualquer
   parte da implementação (agente) invocar `ping`/`traceroute`/binários externos via
   shell, todo input (host, porta) deve ser validado com allowlist estrita e nunca
   concatenado em uma string de shell — preferir libraries nativas (sockets raw/Go
   `net` package) a `exec.Command` com shell.
5. **Falsificação de baseline de anomalias**: um agente comprometido que reporta
   métricas levemente manipuladas por longo período pode "treinar" o baseline
   estatístico a aceitar um estado degradado como normal. Mitigação: baseline
   recalculado com detecção de mudança abrupta suspeita e retenção de dados brutos
   para reprocessamento se uma manipulação for descoberta depois.

## 4. Fora de escopo (aceito, com justificativa)

- Detecção de intrusão em nível de pacote (deep packet inspection) não é feita pelo
  agente/backend — não fazemos captura de payload (Seção 19). Onde IDS/IPS existir,
  é o do próprio UniFi, apenas consumido como evento.
- Proteção contra comprometimento físico do hardware do gateway/AP está fora do
  escopo do software (é responsabilidade de segurança física do site).

## 5. Próximos passos de segurança (não bloqueiam Fase 1, mas devem ser rastreados)

- Definir formalmente o mecanismo de identidade do agente (mTLS vs. credencial
  rotacionável simples) — ver ADR-006.
- ✅ **Resolvido em 2026-08-01 (Fase 8)**: rate limiting nos endpoints de teste
  ativo (`POST /sites/{id}/commands` — ping/dns_lookup/traceroute, Fase 5) — 20
  comandos/min por usuário autenticado (chave por `user_id`, não IP, porque a
  ameaça aqui é abuso de uma conta já autenticada, não tentativa de login
  anônima). Testado de verdade (não só revisão de código): disparando 40
  criações de comando em sequência, o backend retorna 429 antes de completar
  todas.
- ✅ SAST e dependency scanning já rodam no CI desde a Fase 1
  (`security-scan.yml`, `govulncheck` em cada módulo Go).
- ✅ **Resolvido em 2026-08-01 (Fase 5)**: `batch_ping` (novo tipo de comando)
  limita a no máximo 20 alvos por execução (`maxBatchPingTargets` em
  `handlers_commands.go`) — não é uma primitiva de varredura em massa, é uma
  lista pequena e explícita de alvos que o usuário já conhece (ex.: alguns
  APs/servidores), cada um testado individualmente. O rate limiting por
  usuário (acima) se aplica igual a qualquer outro tipo de comando.
- ⏳ **Gap real identificado, ainda não corrigido**: os comandos sob demanda
  (ping/dns_lookup/traceroute/batch_ping) aceitam qualquer `target`/`hostname` informado
  pelo usuário, sem allowlist de RFC 1918 nem confirmação extra para alvos
  externos — a mitigação descrita na Seção 3.1 ("todo alvo precisa estar
  cadastrado... alvo externo exige confirmação explícita") ainda não está
  implementada para essas três ferramentas. Avaliação de risco: aceitável por
  ora porque (a) são operações de alvo único, não primitivas de varredura em
  massa; (b) já têm rate limiting (acima) e auditoria (`requested_by` em
  `agent_commands`); (c) o caso de uso legítimo primário (diagnosticar
  Internet) exige testar hosts externos por natureza — uma restrição
  RFC 1918-only quebraria a funcionalidade central. **Vira bloqueante quando
  um port scanner de verdade for implementado** (Fase 5, ainda não
  começado) — aí sim a mitigação completa (allowlist + confirmação +
  auditoria) é obrigatória antes de expor a funcionalidade.
