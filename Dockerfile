FROM node:24.16.0-alpine@sha256:21f403ab171f2dc89bad4dd69d7721bfd15f084ccb46cdd225f31f2bc59b5c9a AS frontend-build

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY frontend/. .

ENV NODE_ENV=production

RUN npm run build

FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS raven-clone

# renovate: datasource=git-refs depName=https://github.com/kn6plv/Raven
ARG RAVEN_VERSION=main
ARG RAVEN_REF=2173bba55e1a387cfb0c3c11651ce3755c507609

RUN apk add --no-cache git
RUN git clone https://github.com/kn6plv/Raven.git /raven && \
    cd /raven && \
    git checkout "${RAVEN_REF}" && \
    rm -rf /raven/.git

FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS usign-build

RUN apk add --no-cache git build-base cmake
RUN git clone https://git.openwrt.org/project/usign.git /usign-src && \
    cd /usign-src && \
    cmake -B build . && \
    cmake --build build && \
    cp build/usign /usr/bin/usign

FROM ghcr.io/usa-reddragon/mesh-base:main@sha256:966e64f4554d544a518bcdf86fe76769159fda7c63fcb11f4c92f73171720d63

COPY --from=frontend-build /app/dist /www
COPY --from=ghcr.io/usa-reddragon/meshmap-mesh-manager:k8s@sha256:f48c21f842dedeae8b999161fe31374db409995f5f34b9cd665af71ee19351dd /usr/share/nginx/html /meshmap

RUN apk add --no-cache \
    nginx \
    socat \
    iperf3

COPY --from=usign-build /usr/bin/usign /usr/bin/usign

COPY --from=raven-clone /raven /usr/local/raven

# Patch Raven UI to connect WebSocket via nginx proxy instead of directly to port 4404
RUN sed -i 's|`ws://${location.hostname}:4404`|((location.protocol==="https:")?"wss://":"ws://")+location.host+"/raven/ws"|' /usr/local/raven/ui/ui.js
RUN sed -i 's/socket.SOCK_STRAM/socket.SOCK_STREAM/' /usr/local/raven/websocket.uc

COPY --chown=root:root docker/rootfs/. /

# Verify our custom platform.uc implements all exports that upstream Raven expects.
RUN --mount=type=bind,from=raven-clone,source=/raven/platforms/aredn/platform.uc,target=/tmp/raven-upstream-platform.uc \
    sed -n '/^[[:space:]]*return[[:space:]]*{[[:space:]]*$/,/^[[:space:]]*};[[:space:]]*$/p' /tmp/raven-upstream-platform.uc \
    | sed '1d;$d' | tr -d ' ,' | grep -v '^$' | sort > /tmp/raven-upstream-exports.txt && \
    sed -n '/^[[:space:]]*return[[:space:]]*{[[:space:]]*$/,/^[[:space:]]*};[[:space:]]*$/p' /usr/local/raven/platforms/aredn/platform.uc \
    | sed '1d;$d' | tr -d ' ,' | grep -v '^$' | sort > /tmp/raven-custom-exports.txt && \
    if [ ! -s /tmp/raven-upstream-exports.txt ] || [ ! -s /tmp/raven-custom-exports.txt ]; then \
        echo "ERROR: Failed to extract exports from platform.uc (check return-block formatting)." >&2; \
        exit 1; \
    fi && \
    missing=$(comm -23 /tmp/raven-upstream-exports.txt /tmp/raven-custom-exports.txt) && \
    if [ -n "$missing" ]; then \
        echo "ERROR: Custom platform.uc is missing exports required by upstream Raven:" >&2; \
        echo "$missing" >&2; \
        exit 1; \
    fi && \
    extra=$(comm -13 /tmp/raven-upstream-exports.txt /tmp/raven-custom-exports.txt) && \
    if [ -n "$extra" ]; then \
        echo "WARNING: Custom platform.uc has exports not present in upstream Raven:" >&2; \
        echo "$extra" >&2; \
    fi && \
    rm -f /tmp/raven-upstream-exports.txt /tmp/raven-custom-exports.txt

RUN rm -rf /etc/s6/olsrd

COPY mesh-manager /usr/bin/mesh-manager
CMD ["/usr/bin/start.sh"]
