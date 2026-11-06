# Imágenes de producción del backend (SDD 8.4). Un solo Dockerfile, un target por binario:
#
#   docker build -f backend/Dockerfile --target api     -t movo-api     backend
#   docker build -f backend/Dockerfile --target worker  -t movo-worker  backend
#   docker build -f backend/Dockerfile --target migrate -t movo-migrate backend
#
# Nada se compila en el VPS: se construye en la máquina de desarrollo o en CI y se sube a un
# registro. Binarios estáticos sobre una imagen sin shell ni paquetes.

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot AS api
COPY --from=build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker
COPY --from=build /out/worker /app/worker
ENTRYPOINT ["/app/worker"]

# Las migraciones van embebidas en el binario (backend/migrations/embed.go).
FROM gcr.io/distroless/static-debian12:nonroot AS migrate
COPY --from=build /out/migrate /app/migrate
ENTRYPOINT ["/app/migrate"]
