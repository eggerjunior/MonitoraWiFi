# packages/design-tokens — Tokens de design compartilhados

`tokens.json` é a fonte única de verdade (cor claro/escuro, tipografia, espaçamento,
raio, e o mapeamento de `provenance` — rótulo/ícone por fonte de dado, usado no badge
de proveniência descrito em `docs/architecture/02-arquitetura-proposta.md` §2.6).

Nunca editar `tokens.ts`, `tokens.css` ou `DesignTokens.swift` à mão — são gerados:

```bash
node scripts/generate.mjs
```

- `tokens.ts` — consumido pelo `apps/web` (TypeScript).
- `tokens.css` — variáveis CSS (`:root`, `prefers-color-scheme: dark`,
  `[data-theme]`) para uso direto em classes Tailwind/CSS do `apps/web`.
- `DesignTokens.swift` — consumido pelo `apps/ios` (`Color.egger(_:scheme:)` e
  `EggerMetricSource`).

Qualquer mudança de paleta/tipografia/espaçamento começa em `tokens.json`.

## Contraste (WCAG AA)

Toda cor de texto/status em `color.light`/`color.dark` precisa manter
razão de contraste >= 4.5:1 contra `surface` (texto normal, não
"large text" — os textos que usam essas cores no produto, ex.: badge de
status "ONLINE", rodam em `text-xs`/`caption`, abaixo do tamanho que
qualificaria pra exceção de 3:1 do WCAG). Auditoria em 2026-08-02
encontrou 4 cores abaixo do mínimo (`textDisabled` claro e escuro,
`success` e `warning` claro) — ajustadas pro valor mais próximo do
original que ainda cruza 4.5:1 (ver `docs/testing/acessibilidade.md`
pro cálculo e antes/depois completos). Ao alterar qualquer cor aqui,
recalcular o contraste contra `surface` e `background` nos dois temas
antes de commitar.
