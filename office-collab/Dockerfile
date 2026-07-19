FROM docker.io/library/node:24-alpine AS deps
WORKDIR /app
RUN wget -qO /tmp/pnpm.tgz https://registry.npmjs.org/@pnpm/exe/-/exe-11.9.0.tgz \
  && mkdir -p /tmp/pnpm \
  && tar -xzf /tmp/pnpm.tgz -C /tmp/pnpm \
  && mv /tmp/pnpm/package /opt/pnpm \
  && printf '#!/bin/sh\nexec node /opt/pnpm/dist/pnpm.mjs "$@"\n' > /usr/local/bin/pnpm \
  && chmod +x /usr/local/bin/pnpm \
  && rm -rf /tmp/pnpm /tmp/pnpm.tgz
COPY package.json pnpm-lock.yaml* ./
RUN if [ -f pnpm-lock.yaml ]; then pnpm --config.minimumReleaseAge=0 --store-dir .pnpm-store install --frozen-lockfile; else pnpm --config.minimumReleaseAge=0 --store-dir .pnpm-store install; fi

FROM docker.io/library/node:24-alpine
WORKDIR /app
COPY --from=deps /app/node_modules /app/node_modules
COPY package.json server.mjs ./
EXPOSE 1234
CMD ["node", "/app/server.mjs"]
