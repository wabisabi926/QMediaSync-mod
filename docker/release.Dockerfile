# check=skip=SecretsUsedInArgOrEnv
FROM alpine:3.20

ARG TARGETARCH=amd64
ARG TARGETOS=linux

ENV TZ=Asia/Shanghai \
    PATH=/app:$PATH \
    DB_HOST=localhost \
    DB_PORT=5432 \
    DB_USER=qms \
    DB_PASSWORD=qms123456 \
    DB_NAME=qms \
    DB_SSLMODE=disable

RUN apk add --no-cache ca-certificates tzdata inotify-tools postgresql15 su-exec && \
    addgroup -S -g 12331 qms && \
    adduser -S -D -H -u 12331 -G qms qms && \
    mkdir -p /dev/shm /app/scripts && \
    chmod 1777 /dev/shm && \
    chmod 777 /app

WORKDIR /app
COPY --chmod=0755 temp_build/QMediaSync_linux_${TARGETARCH}_exe ./QMediaSync
COPY backend/web_statics ./web_statics/
COPY --chmod=0755 docker/entrypoint.sh ./scripts/docker-entrypoint.sh
COPY --chmod=0755 docker/watch-update.sh ./scripts/watch_update.sh
COPY backend/assets/db_config.html ./web_statics/
COPY backend/icon.ico .

VOLUME ["/app/config", "/media"]
EXPOSE 12333 8095 8094
CMD ["/app/scripts/docker-entrypoint.sh"]
