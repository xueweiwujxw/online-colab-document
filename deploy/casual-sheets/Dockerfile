# syntax=docker/dockerfile:1.7

# Rebuild the exact Casual Sheets release instead of mutating a floating
# runtime image. The recursive checkout also pins the upstream Univer and
# server submodules recorded by the v0.3.4 release.
ARG NODE_VERSION=22-alpine
ARG CASUAL_SHEETS_REF=11182a2f486a5134fa9d1672530df1dc27025ce4

FROM alpine:3.21 AS source
ARG CASUAL_SHEETS_REF
RUN apk add --no-cache git patch
RUN git config --global url."https://github.com/".insteadOf git@github.com: && \
    git clone --recursive https://github.com/CasualOffice/sheets.git /repo && \
    git -C /repo checkout "${CASUAL_SHEETS_REF}" && \
    git -C /repo submodule update --init --recursive
COPY wopi-presence.patch /tmp/wopi-presence.patch
COPY wopi-readonly.patch /tmp/wopi-readonly.patch
COPY server-log-redaction.patch /tmp/server-log-redaction.patch
RUN git -C /repo apply --check /tmp/wopi-presence.patch && \
    git -C /repo apply /tmp/wopi-presence.patch && \
    git -C /repo apply --check /tmp/wopi-readonly.patch && \
    git -C /repo apply /tmp/wopi-readonly.patch && \
    patch --batch --forward -p1 -d /repo/apps/server < /tmp/server-log-redaction.patch

FROM node:${NODE_VERSION} AS build-web
RUN apk add --no-cache bash git python3 make g++ && \
    corepack enable && corepack prepare pnpm@10.33.4 --activate
WORKDIR /repo
COPY --from=source /repo /repo
RUN ./scripts/setup-fork.sh && \
    pnpm install --frozen-lockfile
RUN pnpm --filter @casualoffice/sheets build
ARG VITE_COLLAB_ENABLED=1
ENV VITE_COLLAB_ENABLED=${VITE_COLLAB_ENABLED}
ENV NODE_OPTIONS=--max-old-space-size=4096
RUN pnpm --filter @sheet/web build

FROM node:${NODE_VERSION} AS runtime
RUN apk add --no-cache wget su-exec && \
    corepack enable && corepack prepare pnpm@10.33.4 --activate
ENV NODE_ENV=production \
    HOST=0.0.0.0 \
    PORT=3000
WORKDIR /app
COPY --from=build-web /repo/pnpm-lock.yaml /repo/pnpm-workspace.yaml /repo/package.json ./
COPY --from=build-web /repo/patches patches
COPY --from=build-web /repo/vendor/univer-revamp vendor/univer-revamp
COPY --from=build-web /repo/vendor/design-system vendor/design-system
COPY --from=build-web /repo/apps/web/package.json apps/web/
COPY --from=build-web /repo/apps/server/package.json apps/server/
COPY --from=build-web /repo/packages/sdk/package.json packages/sdk/
RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --prod
COPY --from=build-web /repo/apps/server/src apps/server/src
COPY --from=build-web /repo/apps/server/tsconfig.json apps/server/
COPY --from=build-web /repo/apps/web/dist apps/web/dist
COPY --from=build-web /repo/scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh && \
    mkdir -p /data && chown -R node:node /data
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:3000/health || exit 1
WORKDIR /app/apps/server
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["node", "--import", "tsx", "src/index.ts"]
