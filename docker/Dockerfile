# Stage 1: Build Go backend
FROM golang:1.25-alpine AS go-builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /bin/agent-sandbox ./cmd/sandbox/
RUN CGO_ENABLED=1 go build -o /bin/agent-sandbox-server ./cmd/server/

# Stage 2: Build React frontend
FROM node:22-alpine AS web-builder

WORKDIR /app

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ .
RUN npm run build

# Stage 3: Runtime
FROM alpine:3.20

RUN apk add --no-cache sqlite-libs ca-certificates

COPY --from=go-builder /bin/agent-sandbox /usr/local/bin/
COPY --from=go-builder /bin/agent-sandbox-server /usr/local/bin/
COPY --from=web-builder /app/dist /usr/share/agent-sandbox/web
COPY configs/ /etc/agent-sandbox/configs/

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["agent-sandbox-server"]
CMD ["--port", "8080", "--static-dir", "/usr/share/agent-sandbox/web"]
