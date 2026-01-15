FROM node:24.13.0-alpine@sha256:931d7d57f8c1fd0e2179dbff7cc7da4c9dd100998bc2b32afc85142d8efbc213 AS frontend-build

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY frontend/. .

ENV NODE_ENV=production

RUN npm run build

FROM ghcr.io/usa-reddragon/mesh-base:main@sha256:ecd2d6343483d01d522f5db304459adaa1f3212436662a22aeb15bebdcb5c43f

COPY --from=frontend-build /app/dist /www
COPY --from=ghcr.io/usa-reddragon/meshmap-mesh-manager:k8s@sha256:7a96f1dbcea00a0d71f8c742f98dcdd0dddf5f0e00a38d41e8685bbfc5818cf5 /usr/share/nginx/html /meshmap

RUN apk add --no-cache \
    nginx \
    socat \
    iperf3

COPY --chown=root:root docker/rootfs/. /

RUN rm -rf /etc/s6/olsrd

COPY mesh-manager /usr/bin/mesh-manager
CMD ["/usr/bin/start.sh"]
