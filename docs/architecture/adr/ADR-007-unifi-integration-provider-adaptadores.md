# ADR-007 — UniFiIntegrationProvider com adaptadores plugáveis e capability matrix

## Status
Aceito

## Contexto
Não existe uma única "API do UniFi" estável e universal: há a Network API local
oficial (versionada por console), a Site Manager API oficial (cloud, agregada), SNMP
(cobertura parcial) e uma API não oficial usada pela comunidade (ver
`docs/unifi/capability-matrix.md`). Assumir que todos os endpoints existem em toda
instalação/versão violaria a Seção 4 ("não assumir que todos os endpoints existem").

## Decisão
Criar a interface `UniFiIntegrationProvider` com um adaptador por fonte
(`NetworkAPIAdapter`, `SiteManagerAdapter`, `SNMPAdapter`, `SyslogAdapter`,
`LegacyAdapter` — este último isolado e desativado por padrão). Na configuração da
integração, o agente detecta versão do UniFi OS/Network e popula a capability matrix
daquele `UniFiConsole` (`capabilities: jsonb`). Toda leitura de dado UniFi no
resto do sistema passa por essa camada, nunca chama um endpoint específico
diretamente de fora dela.

## Consequências
- Adicionar suporte a uma nova versão de UniFi ou a um adaptador extra (ex.: um novo
  fabricante, futuramente) não exige tocar em código de domínio/UI — só implementar
  um novo adaptador atrás da mesma interface.
- A UI (iOS/Web) consulta a capability matrix antes de renderizar, permitindo
  degradação honesta ("recurso não suportado nesta versão") em vez de erro ou dado
  inventado.
- Exige investimento inicial maior de design de interface (vs. simplesmente chamar a
  API do UniFi direto do handler) — aceito como custo necessário dado o requisito
  explícito de suportar múltiplos modelos/versões (Seção 1).

## Alternativas consideradas
- **Chamar a Network API local diretamente do backend/worker**: rejeitado — o backend
  não está na LAN do cliente (ver ADR-001); além disso acopla o resto do sistema a uma
  única fonte de dado sem abstração para SNMP/legado/Site Manager.
- **Assumir uma única versão de UniFi suportada oficialmente**: rejeitado — contradiz
  a Seção 1 ("suportar... diferentes gateways; diversos modelos de APs e switches").
