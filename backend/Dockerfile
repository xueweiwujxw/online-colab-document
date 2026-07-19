FROM docker.io/library/golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM docker.io/library/alpine:3.20

WORKDIR /app
COPY --from=build /out/server /app/server
COPY migrations /app/migrations
ENV MIGRATIONS_DIR=/app/migrations
EXPOSE 8080
ENTRYPOINT ["/app/server"]
