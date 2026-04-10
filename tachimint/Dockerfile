FROM node:24-alpine

RUN corepack enable && corepack prepare pnpm@10.33.0 --activate

WORKDIR /app

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .

EXPOSE 5173

CMD ["pnpm", "dev"]
