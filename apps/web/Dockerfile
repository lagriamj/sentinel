FROM node:22.23.2-alpine AS build

WORKDIR /src
COPY package.json package-lock.json ./
RUN npm install -g npm@11.11.0 && npm ci
COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.30.6-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
