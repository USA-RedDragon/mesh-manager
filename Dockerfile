FROM node:24.14.1-alpine@sha256:01743339035a5c3c11a373cd7c83aeab6ed1457b55da6a69e014a95ac4e4700b AS frontend-build

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY frontend/. .

ENV NODE_ENV=production

RUN npm run build

FROM alpine:3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS raven-clone

RUN apk add --no-cache git
RUN git clone https://github.com/kn6plv/Raven.git /raven && \
    cd /raven && \
    TAG=$(git tag --sort=-v:refname | head -1) && \
    if [ -n "$TAG" ]; then \
        echo "Using Raven tag: $TAG"; \
        git checkout "$TAG"; \
    else \
        echo "No tags found, using main branch"; \
    fi && \
    rm -rf /raven/.git

FROM ghcr.io/usa-reddragon/mesh-base:main@sha256:ecd2d6343483d01d522f5db304459adaa1f3212436662a22aeb15bebdcb5c43f

COPY --from=frontend-build /app/dist /www
COPY --from=ghcr.io/usa-reddragon/meshmap-mesh-manager:k8s@sha256:fab7ffbf8f1b7a7b690b39ae21e492021f47052f0fb87336db59792d936d2e27 /usr/share/nginx/html /meshmap

RUN apk add --no-cache \
    nginx \
    socat \
    iperf3

COPY --from=raven-clone /raven /usr/local/raven

COPY --chown=root:root docker/rootfs/. /

RUN rm -rf /etc/s6/olsrd

COPY mesh-manager /usr/bin/mesh-manager
CMD ["/usr/bin/start.sh"]
