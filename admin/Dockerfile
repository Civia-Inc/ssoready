# Development stage
FROM node:22.4 AS development

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY .env tailwind.config.js build.mjs tsconfig.json components.json ./
COPY bin ./bin

# Note: public and src are mounted as volumes in compose.yaml for hot-reloading

EXPOSE 8083

CMD ["npm", "run", "dev"]

# Production stage
FROM node:22.4 AS production

WORKDIR /work

COPY package.json package-lock.json ./
RUN npm ci

# Note: .env should ideally be removed from COPY (pass env vars at runtime instead)
# and components.json may need to be added if the build references it
COPY .env tailwind.config.js build.mjs tsconfig.json ./
COPY bin ./bin
COPY public ./public
COPY src ./src

RUN npm run build-docker

ENTRYPOINT ["npm", "run", "serve-docker"]
