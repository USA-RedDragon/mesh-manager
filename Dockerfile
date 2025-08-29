FROM node:22.18.0-alpine AS frontend-build

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY frontend/. .

ENV NODE_ENV=production

RUN npm run build -- --base=/a/

FROM node:22.18.0-alpine AS new-frontend-build

WORKDIR /app

COPY new-frontend/package.json new-frontend/package-lock.json ./
RUN npm ci --ignore-scripts

COPY new-frontend/. .

ENV NODE_ENV=production

RUN npm run build -- --base=/b/

FROM ghcr.io/usa-reddragon/mesh-base:main@sha256:988256c53c12aad59a2362d4351f6a358a116effac1e8618da25ad37674942b1

COPY --from=frontend-build /app/dist /www
COPY --from=new-frontend-build /app/dist /new-www

RUN apk add --no-cache \
    nginx \
    socat

COPY --chown=root:root docker/rootfs/. /

RUN rm -rf /etc/s6/olsrd

COPY mesh-manager /usr/bin/mesh-manager
CMD ["bash", "/usr/bin/start.sh"]
