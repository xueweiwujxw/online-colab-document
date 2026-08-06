FROM docker.io/casualoffice/sheets:0.3.4

# Casual Sheets serves the native Univer/Hocuspocus application.  Keep the
# upstream runtime intact, but prevent its Fastify request logger from writing
# query-string WOPI/JWT credentials to container logs.
USER root
RUN sed -i "s/Fastify({ logger: true, bodyLimit:/Fastify({ logger: { redact: ['req.url'] }, bodyLimit:/" /app/apps/server/src/index.ts
