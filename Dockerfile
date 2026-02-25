FROM node:24.14.0-alpine@sha256:ffc7bb66190e1732398a70a4642a4e173762c10ed88fdca320a8e2fcc41e5ff2 AS frontend-build

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY frontend/. .

ENV NODE_ENV=production

RUN npm run build

FROM ghcr.io/usa-reddragon/mesh-base:main@sha256:ecd2d6343483d01d522f5db304459adaa1f3212436662a22aeb15bebdcb5c43f

COPY --from=frontend-build /app/dist /www
COPY --from=ghcr.io/usa-reddragon/meshmap-mesh-manager:k8s@sha256:81f9fa6a4fee4f449e4b4716d6b517e07f9d98a73dd0077ddbad2b243c75dda2 /usr/share/nginx/html /meshmap

RUN apk add --no-cache \
    nginx \
    socat \
    iperf3

COPY --chown=root:root docker/rootfs/. /

RUN rm -rf /etc/s6/olsrd

COPY mesh-manager /usr/bin/mesh-manager
CMD ["/usr/bin/start.sh"]
