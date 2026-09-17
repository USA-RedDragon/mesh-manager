FROM node:24.21.0-alpine@sha256:be80f76cf40ec8e42b9bec49f60a55e0660f30af58d3e5a25530785b30ea67e2 AS frontend-build

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY frontend/. .

ENV NODE_ENV=production

RUN npm run build

FROM alpine:3.24.2@sha256:3cf95fe0816180395592b8373f3ec60663f076127617bbacb4eacf9667afe2e9 AS raven-clone

# renovate: datasource=git-refs depName=https://github.com/kn6plv/Raven
ARG RAVEN_VERSION=main
ARG RAVEN_REF=da911006c8dfafef143b4c1d04c2a194ef58b668

RUN apk add --no-cache git
RUN git clone https://github.com/kn6plv/Raven.git /raven && \
    cd /raven && \
    git checkout "${RAVEN_REF}" && \
    rm -rf /raven/.git

FROM alpine:3.24.2@sha256:3cf95fe0816180395592b8373f3ec60663f076127617bbacb4eacf9667afe2e9 AS usign-build

RUN apk add --no-cache git build-base cmake
RUN git clone https://git.openwrt.org/project/usign.git /usign-src && \
    cd /usign-src && \
    cmake -B build . && \
    cmake --build build && \
    cp build/usign /usr/bin/usign

FROM ghcr.io/usa-reddragon/mesh-base:main@sha256:0a93ce39c90863309402238febd3b86332622b319f80a1d643c19deebbd3a15a

COPY --from=frontend-build /app/dist /www
COPY --from=ghcr.io/usa-reddragon/meshmap-mesh-manager:k8s@sha256:9c27837999cbbdbb151dbb82728c6bb4ee91206d65082b578a8d54169efe611a /usr/share/nginx/html /meshmap

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
