# Imagen de producción del frontend (SDD 8.4). Se construye fuera del VPS.
#
#   docker build -f frontend/Dockerfile frontend \
#     --build-arg NEXT_PUBLIC_SUPPORT_WHATSAPP=549351XXXXXXX -t movo-web
#
# Las NEXT_PUBLIC_* se incrustan en el build: cambiarlas exige reconstruir la imagen.
# Si se usa el mapa propio, generar antes frontend/public/map (scripts/map-assets.sh).

FROM node:24.22.0-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG NEXT_PUBLIC_SUPPORT_WHATSAPP=""
ARG NEXT_PUBLIC_MAP_PMTILES_URL=""
ARG NEXT_PUBLIC_MAP_ASSETS_URL=""
ARG NEXT_PUBLIC_MAP_STYLE_URL=""
ENV NEXT_PUBLIC_SUPPORT_WHATSAPP=$NEXT_PUBLIC_SUPPORT_WHATSAPP \
    NEXT_PUBLIC_MAP_PMTILES_URL=$NEXT_PUBLIC_MAP_PMTILES_URL \
    NEXT_PUBLIC_MAP_ASSETS_URL=$NEXT_PUBLIC_MAP_ASSETS_URL \
    NEXT_PUBLIC_MAP_STYLE_URL=$NEXT_PUBLIC_MAP_STYLE_URL \
    NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:24.22.0-alpine AS run
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    HOSTNAME=0.0.0.0 \
    PORT=3000
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
COPY --from=build --chown=node:node /app/public ./public
USER node
EXPOSE 3000
CMD ["node", "server.js"]
