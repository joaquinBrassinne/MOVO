# Fase 7.4: CDN delante del bucket de imágenes

## Objetivo
Servir las fotos de los locales desde un CDN, con caché larga y sin procesar imágenes en Go.
Las subidas siguen yendo siempre al bucket mediante URL prefirmada.

## Qué hace el código
- `IMAGES_CDN_URL` (opcional): base del CDN, por ejemplo `https://cdn.example.com`.
  - Sin definir: las lecturas son URLs prefirmadas del bucket (igual que antes).
  - Definida: la URL pública de lectura es `base + "/" + clave` (sin firma). Cada segmento
    de la clave se escapa para URL.
  - Se valida al arrancar (`config.LoadStorage`): URL absoluta `http`/`https`, con host,
    sin usuario, sin query ni fragmento. Las barras finales se quitan. Si no es válida, la API no arranca.
- El cambio no requiere tocar `cmd/api`: `config.Storage.CDNURL` lo lee `storage.NewS3`.
- Subidas: el presign PUT incluye `Cache-Control: public, max-age=31536000, immutable` y el
  mismo valor viaja en `headers` de la respuesta. El frontend ya envía esos `headers` al subir,
  así que no hay cambio en el cliente.

## Decisión de caché
- Las claves son `businesses/<id>/<128 bits aleatorios><extensión>`, generadas por subida
  (`internal/business/app/commands/images.go`). Una clave no se reutiliza para otra imagen,
  así que el contenido de cada objeto es estable y se puede usar `immutable` durante un año.
- Riesgo conocido: dentro de los 10 minutos de vigencia del presign, el mismo dueño podría
  volver a hacer PUT sobre la misma clave con otro archivo. Un CDN que ya guardó la versión
  anterior la serviría hasta que venza. Es aceptable para el MVP; si se quiere cerrarlo, el
  backend debería rechazar un PUT sobre una clave ya registrada.
- Objetos subidos antes de este cambio no tienen `Cache-Control` guardado. Hay que
  dárselo con una regla del CDN (ver abajo) o con una copia del objeto con metadatos nuevos.

## Requisitos del bucket y del CDN
1. **Bucket público de lectura** (o un origen que el CDN pueda leer sin firma). Las fotos ya son
   públicas en la ficha del local, así que no cambia lo que se expone.
2. **CORS del bucket** (solo para las subidas desde el navegador): el PUT envía
   `Content-Type` y `Cache-Control`. En la consola del proveedor, `AllowedHeaders` debe incluir
   esos encabezados (o `*`). Sin eso el navegador bloquea la subida. `EnsureBucket` en local ya
   pone `*`.
3. **Respuesta del CDN** para `/businesses/...`: `Cache-Control: public, max-age=31536000, immutable`
   (lo que el objeto guarda, o una regla de "Browser TTL" / "Edge TTL" si no se copió el metadato).
4. **Sin cookies ni query en la clave de caché**: la URL es limpia, así que ignorar la query no afecta.
5. Los `GET` de imágenes no necesitan CORS (se muestran con `<img>`).

## Cloudflare R2 con bucket público (opción recomendada)
- Conectar un dominio propio al bucket (`cdn.example.com`) con "Custom Domain". Ese dominio ya
  pasa por la caché de Cloudflare.
- En el bucket, CORS: orígenes del panel, métodos `PUT, GET`, headers `*`, `MaxAge` 600.
- Cache Rule sobre `cdn.example.com/businesses/*`: "Edge TTL" de un año y "Browser TTL" de un año
  (respeta el `Cache-Control` del objeto si está).
- Configurar la API: `S3_ENDPOINT=https://<cuenta>.r2.cloudflarestorage.com`, `S3_REGION=auto`,
  `S3_PATH_STYLE=true`, `IMAGES_CDN_URL=https://cdn.example.com`.

## Amazon CloudFront delante de S3 (alternativa)
- Origen: el bucket S3 con acceso mediante Origin Access Control (OAC). Con OAC el bucket no es
  público, y CloudFront firma la lectura. Si se usa OAC, las URLs sin firma no funcionan:
  hay que usar bucket público o CloudFront con URLs firmadas, y en ese caso `IMAGES_CDN_URL`
  no sirve sin una política adicional. Decidirlo antes de producción.
- Behavior para `/businesses/*`: política de caché con `max-age` de un año (o "Use origin
  cache headers" si el objeto ya trae `Cache-Control`). Compresión activada.
- `IMAGES_CDN_URL` apunta al dominio de la distribución.

## Optimización de imágenes (sin código en Go)
Redimensionar y convertir a WebP se delega en el CDN, no en la API:
- **Cloudflare**: Images Transformations (`/cdn-cgi/image/width=640,format=auto,quality=80/<clave>`).
  Hay que habilitarlo en la zona y usar esa ruta en el frontend para las miniaturas. La URL
  base cambia, así que el frontend lo decide por tamaño.
- **CloudFront**: no hay transformación nativa. Usar una CloudFront Function o Lambda@Edge
  que lea `?w=` y redimensione, o un servicio de imágenes externo. Ese trabajo queda fuera del MVP.
- Mientras no haya transformación, el frontend debe pedir la imagen original con `loading="lazy"`
  y dimensiones fijas en el contenedor.

## Verificación
- `go test ./internal/platform/config/ ./internal/platform/storage/` cubre:
  - `IMAGES_CDN_URL` sin definir, válida, con barras finales, con prefijo de ruta, y las inválidas.
  - Lectura sin CDN (URL firmada del bucket) y con CDN (URL pública, escapada, claves inválidas rechazadas).
  - Subida siempre al bucket, con `Cache-Control` en los headers.
- Prueba manual con CDN: subir una foto, abrir la URL del CDN, comprobar `Cache-Control` y `cf-cache-status: HIT` a la segunda petición.
