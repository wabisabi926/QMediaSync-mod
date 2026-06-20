# check=skip=SecretsUsedInArgOrEnv
# 本地测试专用：apk / npm / Go 全部走国内镜像源，构建产物与 source.Dockerfile 完全一致。
# 用法：docker build -f docker/source.local.Dockerfile -t qmediasync:local .
# 注意：CI/正式构建请用 docker/source.Dockerfile（官方源），此文件勿用于发布。
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-builder
ENV npm_config_registry=https://registry.npmmirror.com

WORKDIR /app
COPY frontend/package*.json ./frontend/
RUN --mount=type=cache,target=/root/.npm cd frontend && npm ci
COPY frontend ./frontend
RUN mkdir -p backend && cd frontend && npm run build

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend-builder
ENV TZ=Asia/Shanghai
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=off \
    CGO_ENABLED=0

RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates git

WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY backend ./
RUN rm -rf ./web_statics
COPY --from=frontend-builder /app/backend/web_statics ./web_statics/
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=v0.0.0
ARG BUILD_DATE=0000-00-00T00:00:00
ARG FANART_API_KEY
ARG TMDB_ACCESS_TOKEN
ARG TMDB_API_KEY
ARG SC_API_KEY

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags "-s -w -X main.Version=${VERSION} -X 'main.PublishDate=${BUILD_DATE}' -X main.FANART_API_KEY=${FANART_API_KEY} -X main.TMDB_ACCESS_TOKEN=${TMDB_ACCESS_TOKEN} -X main.TMDB_API_KEY=${TMDB_API_KEY} -X main.SC_API_KEY=${SC_API_KEY}" -o QMediaSync .

FROM alpine:3.20
ENV TZ=Asia/Shanghai
ENV PATH=/app:$PATH
ENV DB_HOST=localhost
ENV DB_PORT=5432
ENV DB_USER=qms
ENV DB_PASSWORD=qms123456
ENV DB_NAME=qms
ENV DB_SSLMODE=disable

RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.tuna.tsinghua.edu.cn/alpine#g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates tzdata inotify-tools postgresql15 su-exec && \
    addgroup -S -g 12331 qms && \
    adduser -S -D -H -u 12331 -G qms qms && \
    mkdir -p /dev/shm /app/scripts && \
    chmod 1777 /dev/shm && \
    chmod 777 /app

WORKDIR /app
COPY --from=backend-builder --chmod=0755 /app/backend/QMediaSync ./QMediaSync
COPY --from=backend-builder /app/backend/web_statics ./web_statics/
COPY --chmod=0755 docker/entrypoint.sh ./scripts/docker-entrypoint.sh
COPY --chmod=0755 docker/watch-update.sh ./scripts/watch_update.sh
COPY backend/icon.ico ./icon.ico

VOLUME ["/app/config", "/media"]
EXPOSE 12333
EXPOSE 8095
EXPOSE 8094
CMD ["/app/scripts/docker-entrypoint.sh"]
