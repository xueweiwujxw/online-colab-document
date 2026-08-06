FROM docker.io/oven/bun:1.3.14 AS web
ARG CASUAL_DOCS_REF=d11605185698cfc4b16a83a975cfecc8056ac348
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN git clone https://github.com/CasualOffice/docs.git . \
 && git checkout "$CASUAL_DOCS_REF" \
 && git config --global url."https://github.com/".insteadOf git@github.com: \
 && git submodule update --init --recursive
COPY wopi-markdown.patch /tmp/wopi-markdown.patch
RUN git apply --check /tmp/wopi-markdown.patch && git apply /tmp/wopi-markdown.patch
WORKDIR /src/docx-editor
RUN bun install --frozen-lockfile
ENV VITE_COLLAB_ENABLED=true
RUN bun run build && bun run build:demo

FROM docker.io/library/node:22 AS run
RUN groupadd -r casual && useradd -r -g casual casual
WORKDIR /app/collab
COPY --from=web /src/collab/package.json ./
RUN npm install --omit=dev
COPY --from=web /src/collab/ ./
RUN sed -i "s/logger: true,/logger: { redact: ['req.url'] },/" /app/collab/src/index.ts
COPY --from=web /src/docx-editor/examples/vite/dist /app/web/dist
ENV PORT=3000 HOST=0.0.0.0 CASUAL_FILE_EXT=.docx
USER casual
EXPOSE 3000
CMD ["npm", "start"]
