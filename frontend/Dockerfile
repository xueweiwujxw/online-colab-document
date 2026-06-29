FROM docker.io/library/node:24-alpine AS deps

WORKDIR /app
RUN wget -qO /tmp/pnpm.tgz https://registry.npmjs.org/@pnpm/exe/-/exe-11.9.0.tgz \
  && mkdir -p /tmp/pnpm \
  && tar -xzf /tmp/pnpm.tgz -C /tmp/pnpm \
  && mv /tmp/pnpm/package /opt/pnpm \
  && printf '#!/bin/sh\nexec node /opt/pnpm/dist/pnpm.mjs "$@"\n' > /usr/local/bin/pnpm \
  && chmod +x /usr/local/bin/pnpm \
  && rm -rf /tmp/pnpm /tmp/pnpm.tgz
COPY package.json pnpm-lock.yaml* pnpm-workspace.yaml .npmrc ./
RUN if [ -f pnpm-lock.yaml ]; then pnpm --config.minimumReleaseAge=0 --store-dir .pnpm-store install --frozen-lockfile; else pnpm --config.minimumReleaseAge=0 --store-dir .pnpm-store install; fi

FROM deps AS build

COPY . .
ARG VITE_API_BASE_URL=http://localhost:8080
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
RUN pnpm --config.verify-deps-before-run=false run build

FROM docker.io/library/node:24-alpine

WORKDIR /app
COPY --from=build /app/dist /app/dist
COPY server.mjs /app/server.mjs
EXPOSE 3000
CMD ["node", "/app/server.mjs"]
