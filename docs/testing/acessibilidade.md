# Acessibilidade (Fase 8)

Primeira auditoria dedicada, 2026-08-02 — antes disso só revisão de código
pontual ao longo das fases (roadmap). Cobre WCAG (web) e VoiceOver/Dynamic
Type (iOS, via revisão de código — sem dispositivo físico disponível neste
ambiente de desenvolvimento, mesma limitação já documentada para
`apps/ios/README.md`).

## Web — contraste (WCAG AA, 4.5:1 pra texto normal)

Calculado com a fórmula de luminância relativa do WCAG 2 contra cada cor
de texto/status em `packages/design-tokens/tokens.json`, nos dois temas,
contra `surface` e `background`.

**Antes** (4 falhas reais):

| Token | Tema | Contra | Antes | Depois | Correção |
|---|---|---|---|---|---|
| `textDisabled` | claro | surface | 2.54 | 4.52 | `#9CA3AF` → `#6D7787` |
| `textDisabled` | claro | background | 2.35 | — | (mesma cor, ratio sobe junto) |
| `textDisabled` | escuro | surface | 2.82 | 4.55 | `#5B6270` → `#7B8394` |
| `textDisabled` | escuro | background | 3.09 | 4.97 | (mesma cor) |
| `success` | claro | surface | 4.44 | 4.52 | `#0F8A3F` → `#0F893E` |
| `warning` | claro | surface | 3.81 | 4.57 | `#B5750B` → `#A3690A` |

`textDisabled` era usado em texto real (ex.: "Detalhe de rádio... ainda
não confirmado", rodapés de proveniência) em `text-xs`/`caption` — abaixo
do tamanho que qualificaria pra exceção de "large text" (3:1) do WCAG, não
apenas decoração. `success`/`warning` (claro) são usados em badges de
status (`ONLINE`, "Atenção") também em texto pequeno. As 4 cores foram
ajustadas pro valor mais próximo do original (mesmo matiz) que ainda cruza
4.5:1 — mudança visualmente mínima, não uma repaginação de paleta.
`accent`, `critical`, `success`/`warning` (escuro), `textPrimary`,
`textSecondary` já cruzavam 4.5:1 nos dois temas e não foram tocados.

Tokens regenerados (`node packages/design-tokens/scripts/generate.mjs`) —
`tokens.ts`, `tokens.css`, `DesignTokens.swift` todos derivados de
`tokens.json`, nunca editados à mão.

## Web — outros achados reais

- **`Sidebar.tsx` recolhido**: cada item de navegação virava só a
  primeira letra do rótulo (`item.label.slice(0, 1)`) sem `aria-label` —
  só `title` (que leitores de tela não tratam como nome acessível
  confiável). "Clientes" e "Configurações" colidiam no mesmo "C",
  indistinguíveis tanto visualmente quanto por leitor de tela. Corrigido
  com `aria-label={item.label}` sempre que recolhido.
- **`login/page.tsx`**: os dois `<input>` (e-mail/senha) removiam o anel
  de foco padrão do navegador (`outline-none`) e dependiam só de
  `focus:border-egg-accent` (mudança de cor da borda) — indicador de foco
  fraco pra baixa visão/daltonismo (WCAG 2.4.7/2.4.11). Substituído por
  `focus:outline focus:outline-2 focus:outline-egg-accent
  focus:outline-offset-1`, mantendo a borda. Nenhum outro campo do
  produto usava `outline-none` — era um caso isolado.
- Já conforme sem mudança: `lang="pt-BR"` no `<html>`, nenhum `<img>` sem
  `alt` (não há `<img>` bruto em lugar nenhum), nenhum elemento clicável
  fora de `<button>`/`<Link>` reais, todo `<input>` com `<label
  htmlFor>` associado, `role="alert"` já usado na mensagem de erro de
  login. `eslint-config-next/core-web-vitals` já inclui as regras
  recomendadas de `eslint-plugin-jsx-a11y` — `npm run lint` continua
  verde após todas as mudanças.

## iOS — revisão de código (sem dispositivo físico)

- **Dynamic Type**: nenhuma ocorrência de `.font(.system(size: <número>))`
  (tamanho fixo em pontos, não escala) em todo `apps/ios/Sources` — todo
  texto usa estilos semânticos (`.caption`, `.headline`, `.body`,
  `.system(.caption, design: .monospaced)`, que ainda aceita um
  `Font.TextStyle` como base e escala normalmente). Nenhum
  `.frame(width:)` fixo que pudesse cortar texto em tamanhos de
  acessibilidade maiores.
- **VoiceOver**: nenhum ícone isolado (`Image(systemName:)` sozinho) em
  lugar nenhum — todo ícone aparece via `Label(texto, systemImage:)`,
  cujo nome acessível já é o texto, nunca um controle "mudo" pro
  VoiceOver. Toda tela usa `List`/`Section`/`LabeledContent`, que o
  SwiftUI já agrupa e anuncia de forma sensata por padrão.
- **Limitação real, não resolvida**: nenhuma verificação foi feita com
  VoiceOver de verdade rodando (ordem de leitura, agrupamento de
  elementos compostos como `AnomalyRow`) — este ambiente de
  desenvolvimento não tem Xcode/simulador com VoiceOver nem um
  dispositivo físico, mesma limitação já documentada em
  `apps/ios/README.md` pra testes unitários. Pendência real, não
  esquecida: validar com Xcode/dispositivo real antes de qualquer
  auditoria de acessibilidade se considerar "fechada".

## O que não foi feito (fora de escopo desta auditoria)

- Certificação WCAG formal (auditoria completa exigiria ferramenta
  dedicada — ex.: axe-core rodando contra o site real — e revisão manual
  de navegação por teclado em cada uma das ~20 rotas, não só as
  encontradas nesta revisão pontual).
- Teste com leitor de tela real no web (NVDA/JAWS/VoiceOver macOS).
- Qualquer verificação de VoiceOver/Dynamic Type do iOS em hardware real.
