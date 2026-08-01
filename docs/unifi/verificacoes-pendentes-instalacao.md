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
3. ⏳ **Pendente** — usuário não sabe de cabeça. Próximo passo: buscar "API"
   na caixa "Pesquisar Configurações" do console (visível no print) ou
   checar `Settings → Control Plane → Integrations` / ícone do usuário no
   canto superior direito — versões recentes do UniFi Network (10.x)
   geralmente expõem geração de API key de "local application" numa dessas
   duas telas.
4. ⏳ **Pendente** — depende da resposta ao item 3 (a tela de API/Integrations
   normalmente já diz se o método é API key, usuário/senha, ou os dois).
5. **Parcialmente confirmado em 2026-08-01**: existe conta Ubiquiti vinculada
   (login ativo em `unifi.ui.com`, visível no print). Usuário ainda não
   decidiu se quer habilitar a Site Manager API (cloud, opcional) ou operar
   100% self-hosted — decisão adiada, não bloqueia o restante do
   levantamento.

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
