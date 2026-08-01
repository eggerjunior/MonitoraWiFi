# ADR-003 — Go como stack unificada de backend, worker e agente

## Status
Aceito

## Contexto
O documento-fonte já indica preferência por Go para backend, worker e agente local
(Seções 3 e 5), e o agente precisa ser um binário leve, multiplataforma
(Linux amd64/arm64, macOS), sem runtime pesado, capaz de manipular sockets de baixo
nível (ICMP, ARP, SNMP) com boa performance e footprint de memória pequeno.

## Decisão
Backend (`apps/api`), worker (`apps/worker`) e agente local (`apps/local-agent`) são
implementados em Go, compartilhando o máximo possível de código de domínio via
módulos internos (não necessariamente `packages/network-models`, que é a
representação compartilhada *conceitual*/contratual, também usada por Swift/TS).

## Consequências
- Um único toolchain de build/teste/lint para os três componentes de servidor,
  simplificando CI (Seção 22).
- Cross-compilation nativa do Go facilita gerar binários `linux/amd64`,
  `linux/arm64` e `darwin` do agente a partir do mesmo pipeline, sem toolchain extra.
- Bibliotecas maduras em Go para os protocolos exigidos (ICMP, DNS, SNMP, mDNS,
  syslog) reduzem a necessidade de escrever parsers de protocolo do zero.
- Equipe precisa de proficiência em Go para os três componentes — aceito, dado que é
  também a escolha nativa mais adequada tecnicamente para o agente (não uma
  concessão).

## Alternativas consideradas
- **Agente em Rust**: melhor para controle de memória/segurança de baixo nível, mas
  aumenta a superfície de linguagens do projeto sem benefício claro dado que Go já
  atende aos requisitos de binário leve e cross-compilation; reavaliar apenas se
  surgir necessidade de performance que Go não atenda.
- **Agente em Python**: rejeitado — runtime pesado para instalação "curl | sh" leve,
  dependência de interpretador no host do cliente, pior para binário único distribuído.
