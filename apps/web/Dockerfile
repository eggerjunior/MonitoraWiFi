# syntax=docker/dockerfile:1
# Build context: raiz do monorepo (precisa de packages/design-tokens).
FROM node:20-slim AS build
WORKDIR /repo
COPY apps/web/package.json apps/web/package-lock.json apps/web/
RUN cd apps/web && npm ci
COPY packages/design-tokens /repo/packages/design-tokens
COPY apps/web /repo/apps/web
WORKDIR /repo/apps/web
RUN npm run build

FROM node:20-slim AS runtime
WORKDIR /repo
ENV NODE_ENV=production
COPY --from=build /repo/packages/design-tokens /repo/packages/design-tokens
COPY --from=build /repo/apps/web/package.json /repo/apps/web/package-lock.json /repo/apps/web/
WORKDIR /repo/apps/web
RUN npm ci --omit=dev
COPY --from=build /repo/apps/web/.next ./.next
COPY --from=build /repo/apps/web/public ./public
COPY --from=build /repo/apps/web/next.config.ts ./next.config.ts
EXPOSE 3000
CMD ["npx", "next", "start", "-p", "3000"]
