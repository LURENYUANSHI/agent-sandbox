# Stage 1: Build Go backend
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /bin/agent-sandbox ./cmd/sandbox
RUN CGO_ENABLED=1 go build -o /bin/agent-sandbox-server ./cmd/server

# Stage 2: Build React frontend
FROM node:22-alpine AS web-builder

WORKDIR /app/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ .
RUN npm run build

# Stage 3: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app

COPY --from=go-builder /bin/agent-sandbox /usr/local/bin/
COPY --from=go-builder /bin/agent-sandbox-server /usr/local/bin/
COPY --from=web-builder /app/web/dist /app/web/dist
COPY configs/ /app/configs/

EXPOSE 8080

ENTRYPOINT ["agent-sandbox-server"]
