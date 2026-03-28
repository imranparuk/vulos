# Vula OS — Container
#
# Build: docker build -t vulos .
# Run:   docker run -p 8080:8080 vulos
# Open:  http://localhost:8080

FROM node:22-alpine AS frontend
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY index.html vite.config.js eslint.config.js ./
COPY src/ src/
COPY public/ public/
RUN npm run build

FROM golang:alpine AS backend
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY backend/ .
RUN go mod download && go build -ldflags="-s -w" -o /vulos-server ./cmd/server

FROM alpine:edge

# Core packages + Mozilla CA bundle
RUN apk add --no-cache python3 curl jq ca-certificates \
    && curl -fsSL https://curl.se/ca/cacert.pem -o /etc/ssl/certs/ca-certificates.crt

# Docker CLI for launching neko browser container
RUN apk add --no-cache docker-cli 2>/dev/null || \
    echo "Docker CLI not available — browser will try native neko binary"

RUN mkdir -p /opt/vulos/webroot /opt/vulos/apps \
    /var/lib/vulos /root/.vulos/data /root/.vulos/db /root/.vulos/sandbox \
    /tmp/xdg-runtime

COPY --from=backend /vulos-server /usr/local/bin/vulos-server
COPY --from=frontend /app/dist /opt/vulos/webroot
COPY apps/ /opt/vulos/apps/

RUN touch /var/lib/vulos/.setup-complete

ENV PORT=8080
ENV AI_PROVIDER=ollama
ENV AI_ENDPOINT=http://host.docker.internal:11434
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
ENV XDG_RUNTIME_DIR=/tmp/xdg-runtime
ENV WLR_BACKENDS=headless
ENV WLR_RENDERER=pixman

EXPOSE 8080
CMD ["/usr/local/bin/vulos-server", "-env", "local"]
