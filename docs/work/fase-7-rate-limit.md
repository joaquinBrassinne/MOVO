# Fase 7.3 — Rate limiting distribuido con Redis (nota de trabajo)

**Objetivo:** que los límites de login, de registro/reseteo de clave y de cupones cuenten en común entre varias instancias de la API. Antes, cada proceso contaba por su cuenta y con N instancias el límite se multiplicaba por N.

## Qué cambió
- `internal/platform/cache/limiter.go`: `FixedWindow`, un contador de ventana fija por clave en Redis. Tres operaciones atómicas (scripts Lua): `Allow` (solo lee), `Attempt` (cuenta, y no cuenta cuando ya está agotado) y `Fail` (cuenta un fallo). Además `Reset` (borra la clave).
- `internal/identity/adapters/ratelimit/redis.go` y `internal/benefits/adapters/ratelimit/redis.go`: `RedisLimiter`, la versión Redis de los límites en memoria. Cumplen los mismos puertos (`identity.LoginLimiter`, `identity.RegistrationLimiter`, `benefits.GenerationLimiter`) sin cambiar ninguno.
- `cmd/api`: la conexión a Redis se abre una sola vez (`openRedis`) y la comparten la caché de búsqueda y los límites. Sin `REDIS_URL` se usan los límites en memoria, como antes.
- Los constructores `newIdentityRoutes` y `newBenefitsRoutes` quedan igual para los tests; las versiones `...Shared` reciben el store de Redis.

## Decisiones
- **Mismo comportamiento que en memoria:** la ventana de cada clave empieza con su primer intento contado y dura lo mismo (no se extiende con fallos posteriores); el login cuenta solo los fallos y se limpia en éxito (`Reset`); el registro y los cupones cuentan cada intento y, agotado el cupo, rechazan sin contar más. `retryAfter` sale del TTL de la clave.
- **Scripts Lua:** el conteo y la expiración van en un mismo comando atómico; una clave nunca queda sin TTL (la expiración se fija al crearla, o si le falta). Dos instancias compitiendo no pueden pasar el cupo.
- **Claves con hash:** se guarda `SHA-256` del identificador (email o IP), no el valor en claro, en línea con la regla de no guardar datos personales.
- **Fail open:** cada operación tiene un tope de 300 ms (`cache.LimiterTimeout`). Si Redis falla, la petición pasa y se registra un `WARN` con el nombre del límite, nunca la clave. Un login o un cupón no se bloquean por Redis caído.
- **`MaxKeys` no aplica en Redis:** cada contador vence con su ventana, así que la memoria queda acotada sin tope de claves.
- **Reloj:** el tiempo lo marca Redis, no el reloj inyectado de la versión en memoria. Por eso los tests de ventana usan `miniredis.FastForward`.
- **Un límite por namespace:** `ratelimit:login`, `ratelimit:signup` y `ratelimit:coupon` tienen presupuestos separados, igual que hoy.

## Cómo verificar con dos instancias
1. Levantar Redis (`make up` o un contenedor `redis`) y definir `REDIS_URL=redis://localhost:6379`.
2. Arrancar dos instancias de la API en puertos distintos, por ejemplo `HTTP_ADDR=:8080` y `HTTP_ADDR=:8081`, con la misma base y el mismo `REDIS_URL`.
3. Probar el login con una contraseña incorrecta: cinco fallos repartidos entre `:8080` y `:8081` deben bloquear el sexto intento en ambas, con `429` y `Retry-After`.
4. Probar los cupones con `COUPON_RATE_LIMIT_ATTEMPTS` (por ejemplo 3) repartiendo las solicitudes entre las dos instancias: la cuarta se rechaza en las dos.
5. Apagar Redis y repetir un login: la API responde normalmente y el log muestra `rate limit unavailable, failing open`.

Con `REDIS_URL` vacío, las dos instancias no comparten nada: el límite vuelve a ser por proceso.

## Verificación
- `go build ./...` y `go vet` sobre `cmd`, `internal/platform/cache`, `internal/identity` e `internal/benefits`: sin errores.
- `go test` de `internal/platform/cache`, `internal/identity/adapters/ratelimit`, `internal/benefits/adapters/ratelimit`, `internal/platform/config` y `cmd/api`: todos pasan.
- `-race` no se pudo correr en este entorno (requiere cgo).

## Pendiente
- Los tests de integración con dos instancias reales y un Redis de verdad siguen siendo manuales (pasos de arriba).
- El comentario de `internal/benefits/ports.go` (`GenerationLimiter`) todavía dice que "el puerto deja que Redis lo reemplace". No se tocó porque el archivo está fuera de la superficie de esta tarea.
