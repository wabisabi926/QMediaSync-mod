FROM node:22-alpine AS frontend-builder

WORKDIR /app
COPY frontend/package*.json ./frontend/
RUN cd frontend && npm ci
COPY frontend ./frontend
RUN mkdir -p backend && cd frontend && npm run build

FROM golang:1.25-alpine AS backend-builder
ENV TZ=Asia/Shanghai
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=off \
    CGO_ENABLED=0

RUN apk add --no-cache ca-certificates git

WORKDIR /app/backend
COPY backend/go.mod ./
RUN go mod download
COPY backend ./
COPY --from=frontend-builder /app/backend/web_statics ./web_statics/
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=v0.0.0
ARG BUILD_DATE=0000-00-00T00:00:00
ARG FANART_API_KEY
ARG DEFAULT_TMDB_ACCESS_TOKEN
ARG DEFAULT_TMDB_API_KEY
ARG DEFAULT_SC_API_KEY
ARG ENCRYPTION_KEY

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags "-s -w -X main.Version=${VERSION} -X 'main.PublishDate=${BUILD_DATE}' -X main.FANART_API_KEY=${FANART_API_KEY} -X main.DEFAULT_TMDB_ACCESS_TOKEN=${DEFAULT_TMDB_ACCESS_TOKEN} -X main.DEFAULT_TMDB_API_KEY=${DEFAULT_TMDB_API_KEY} -X main.DEFAULT_SC_API_KEY=${DEFAULT_SC_API_KEY} -X main.ENCRYPTION_KEY=${ENCRYPTION_KEY}" -o QMediaSync .

FROM alpine:3.20
ENV TZ=Asia/Shanghai
ENV PATH=/app:$PATH
ENV DB_HOST=localhost
ENV DB_PORT=5432
ENV DB_USER=qms
ENV DB_PASSWORD=qms123456
ENV DB_NAME=qms
ENV DB_SSLMODE=disable

RUN apk add --no-cache ca-certificates tzdata ffmpeg inotify-tools postgresql15 && \
    addgroup -S -g 12331 qms && \
    adduser -S -D -H -u 12331 -G qms qms && \
    mkdir -p /dev/shm /app/scripts && \
    chmod 1777 /dev/shm

WORKDIR /app
COPY --from=backend-builder /app/backend/QMediaSync ./QMediaSync
COPY --from=backend-builder /app/backend/web_statics ./web_statics/
COPY docker/entrypoint.sh ./scripts/docker-entrypoint.sh
COPY docker/watch-update.sh ./scripts/watch_update.sh
COPY backend/icon.ico ./icon.ico

RUN chmod +x /app/scripts/docker-entrypoint.sh /app/scripts/watch_update.sh /app/QMediaSync && \
    chmod 777 /app

VOLUME ["/app/config", "/media"]
EXPOSE 12333
EXPOSE 8095
EXPOSE 8094
CMD ["/app/scripts/docker-entrypoint.sh"]
