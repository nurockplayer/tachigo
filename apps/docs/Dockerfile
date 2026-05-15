# dev only - not for production
FROM node:24.14.1-alpine

RUN addgroup -S app && adduser -S app -G app
RUN corepack enable && corepack prepare pnpm@10.33.0 --activate

WORKDIR /workspace
ENV CI=1

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/docs/package.json apps/docs/package.json
COPY apps/dashboard/package.json apps/dashboard/package.json
COPY apps/extension/package.json apps/extension/package.json
COPY packages/api-client/package.json packages/api-client/package.json
COPY packages/shared-types/package.json packages/shared-types/package.json
RUN pnpm install --frozen-lockfile --ignore-scripts --filter ./apps/docs

COPY --chown=app:app apps/docs apps/docs
COPY --chown=app:app docs docs

USER app
EXPOSE 3000

CMD ["pnpm", "--filter", "./apps/docs", "start", "--host", "0.0.0.0", "--port", "3000"]
