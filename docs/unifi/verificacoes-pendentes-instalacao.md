# Verificações Pendentes na Instalação UniFi Real

Estes itens **não podem ser confirmados a partir de documentação pública** — exigem
acesso direto ao console (Cloud Gateway Max) da instalação Egger. Nenhum deles deve
ser assumido/simulado antes de resposta real; a Fase 3 (integração UniFi) trata cada
"a validar" da capability matrix como bloqueado até esta lista ser respondida.

## Versões e disponibilidade de API

1. Qual a versão exata do **UniFi OS** rodando no Cloud Gateway Max?
2. Qual a versão exata do **UniFi Network** (aplicação, não o SO)?
3. A **Network API local** já vem habilitada por padrão nessa versão, ou exige
   ativação manual/gerar API key pelo console?
4. Qual o método de autenticação disponível para a Network API local nessa versão
   específica: usuário/senha local, API key de "local application", ou ambos?
5. Existe uma conta Ubiquiti (`unifi.ui.com`) vinculada a este console? Se sim, a
   **Site Manager API** está acessível e é desejável habilitá-la (opcional, cloud) ou
   o requisito é operar 100% self-hosted sem depender da nuvem Ubiquiti?

## Campos realmente expostos pela Network API local nesta versão

6. Para os APs U7 Pro: a resposta da API inclui, por rádio, canal, largura de canal,
   potência de transmissão, utilização do canal, número de clientes, airtime,
   retries e PHY rate — ou apenas um subconjunto?
7. Para o Switch Lite 16 PoE: a API expõe estatística por porta (RX/TX, erros, CRC,
   flaps, consumo PoE em watts, orçamento PoE total) nesta versão?
8. Eventos/alarmes (Seção 4.1 "Eventos e alarmes"): são entregues via polling da API,
   via webhook nativo do UniFi, ou só observáveis via syslog do console?
9. Topologia via LLDP entre dispositivos UniFi: disponível via API nesta versão, ou
   precisa ser inferida (porta do switch reportada pelo AP + inventário)?
10. DPI (Deep Packet Inspection / categorização de aplicação por cliente) está
    habilitado neste console? Se sim, quais campos de categoria/aplicação a API
    expõe por cliente?

## Configuração de rede da instalação (para validar contra o modelo de dados)

11. Confirmar que as sub-redes declaradas no documento-fonte
    (`Egger_Principal 192.168.110.0/24`, `Egger_IoT VLAN 120 192.168.120.0/24`,
    `Egger_Convidados VLAN 130 192.168.130.0/24`) correspondem exatamente à
    configuração atual no console (nomes, IDs de VLAN e CIDRs podem ter sido
    ajustados desde a redação do documento-fonte).
12. IPv6 está habilitado em alguma dessas redes? Em qual modo (prefixo delegado,
    NAT66, desabilitado)?
13. Existe VPN configurada no gateway (Seção 4.1 "Estado de VPNs")? Qual tipo
    (WireGuard/OpenVPN/IPsec nativo do UniFi) e é usada para acesso remoto de
    administração?
14. Existem regras de firewall/IDS-IPS já habilitadas que precisam ser
    consideradas como eventos de segurança a ingerir desde o início (Seção 4.1)?

## Hardware e licenciamento

15. As duas WANs (primária e secundária) estão ambas fisicamente conectadas e
    configuradas para failover/balanceamento no Cloud Gateway Max, ou a secundária
    é um cenário planejado a confirmar?
16. Confirmar o firmware atual de cada um dos 4 APs U7 Pro e do Switch Lite 16 PoE
    (relevante para saber se recursos Wi-Fi 7 relevantes, como MLO, já estão
    disponíveis na versão de firmware instalada).
17. SNMP está habilitado no console? Em qual versão (v2c/v3) e com quais
    credenciais/community string (não registrar o valor aqui — apenas confirmar que
    existe e onde a credencial será armazenada).
18. Syslog está configurado para ser enviado a algum coletor hoje? Se sim, qual
    formato/porta, para o agente poder atuar como receiver compatível.

## Processo de validação recomendado

Cada resposta obtida deve:
1. Atualizar a linha correspondente em `docs/unifi/capability-matrix.md` de
   "a validar" para "confirmado" ou "indisponível", com a data da verificação.
2. Quando a resposta implicar mudança no modelo de dados
   (`docs/architecture/05-modelo-dados.md`), registrar isso como ajuste antes de
   escrever a migração definitiva da Fase 3.
3. Quando a resposta revelar uma limitação not documentada aqui, adicionar uma nova
   linha neste arquivo e no threat model, se aplicável.
