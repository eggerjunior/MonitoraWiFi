# apps/web — Egger Network Intelligence (Web)

Status: Fase 1 (shell funcional). Next.js 16 (App Router), TypeScript estrito,
Tailwind CSS v4, React 19.

## Desenvolvimento

```bash
npm install
npm run dev
```

Requer o backend (`apps/api`) rodando e acessível em `API_BASE_URL`
(padrão `http://localhost:8080/api/v1`, ver `src/lib/session.ts`).

## O que existe nesta fase

- Login (`/login`) contra o backend real via rota BFF (`/api/auth/login`,
  `/api/auth/logout`) — o navegador nunca fala diretamente com `apps/api`;
  o cookie de sessão é re-emitido sob o domínio do próprio Next.js.
- Shell de navegação (`(dashboard)/layout.tsx`): sidebar recolhível, tema
  claro/escuro (persistido em `localStorage`, sem flash — ver
  `node_modules/next/dist/docs/01-app/02-guides/preventing-flash-before-hydration.md`),
  gate de autenticação real (Server Component consultando `/auth/me`).
- Visão geral (`/`) lista organizações/sites reais vindos do backend.
- Demais módulos (Internet, Wi-Fi, Dispositivos, Clientes, Mapa, Diagnósticos,
  Alertas, Relatórios, Configurações) são placeholders honestos — sem dado
  simulado — até suas fases correspondentes no roadmap.

## Design tokens

Cores/tipografia vêm de `packages/design-tokens/tokens.css` (importado em
`src/app/globals.css`) — nunca hardcoded aqui. Ver `packages/design-tokens/README.md`.

## Nota sobre esta versão do Next.js

Este projeto usa Next.js 16, que renomeou Middleware para **Proxy**
(`proxy.ts`) e tem outras mudanças relevantes desde versões anteriores.
Antes de alterar convenções de roteamento, cache ou autenticação, consulte
`node_modules/next/dist/docs/` (instalado localmente) em vez de assumir
comportamento de versões anteriores.
