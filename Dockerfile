FROM node:24.12.0-alpine AS frontend-build

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY frontend/. .

ENV NODE_ENV=production

RUN npm run build -- --base=/a/

FROM node:24.12.0-alpine@sha256:c921b97d4b74f51744057454b306b418cf693865e73b8100559189605f6955b8 AS new-frontend-build

WORKDIR /app

COPY new-frontend/package.json new-frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY new-frontend/. .

ENV NODE_ENV=production

RUN npm run build -- --base=/b/

FROM ghcr.io/usa-reddragon/mesh-base:main@sha256:ecd2d6343483d01d522f5db304459adaa1f3212436662a22aeb15bebdcb5c43f

COPY --from=frontend-build /app/dist /www
COPY --from=new-frontend-build /app/dist /new-www
COPY --from=ghcr.io/usa-reddragon/meshmap-mesh-manager:k8s@sha256:d2a340c8be510ae2fc4746ced48d3ee6394c2d918f93e554a46443bac50664ba /usr/share/nginx/html /meshmap

RUN apk add --no-cache \
    nginx \
    socat \
    iperf3

COPY --chown=root:root docker/rootfs/. /

RUN rm -rf /etc/s6/olsrd

COPY mesh-manager /usr/bin/mesh-manager
CMD ["/usr/bin/start.sh"]
