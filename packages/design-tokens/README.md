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
