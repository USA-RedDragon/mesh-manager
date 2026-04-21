FROM node:24.14.1-alpine@sha256:8510330d3eb72c804231a834b1a8ebb55cb3796c3e4431297a24d246b8add4d5 AS frontend-build

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY frontend/. .

ENV NODE_ENV=production

RUN npm run build

FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS raven-clone

# renovate: datasource=git-refs depName=https://github.com/kn6plv/Raven
ARG RAVEN_VERSION=main
ARG RAVEN_REF=861f355a0b1d032e34469c01b875d8d5307e66a8

RUN apk add --no-cache git
RUN git clone https://github.com/kn6plv/Raven.git /raven && \
    cd /raven && \
    git checkout "${RAVEN_REF}" && \
    rm -rf /raven/.git

FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS usign-build

RUN apk add --no-cache git build-base cmake
RUN git clone https://git.openwrt.org/project/usign.git /usign-src && \
    cd /usign-src && \
    cmake -B build . && \
    cmake --build build && \
    cp build/usign /usr/bin/usign

FROM ghcr.io/usa-reddragon/mesh-base:main@sha256:35ba2c653fad5022d13bb33a6bce4937988ca591cb7a117b3cb91cf8b7eb0b6a

COPY --from=frontend-build /app/dist /www
COPY --from=ghcr.io/usa-reddragon/meshmap-mesh-manager:k8s@sha256:20c386ca5237a3dd65e7fa394f9fac60d27cfd14a217b5a1395862f17f66216b /usr/share/nginx/html /meshmap

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
