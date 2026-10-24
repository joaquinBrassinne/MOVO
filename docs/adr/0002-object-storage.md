# 0002 — Object storage: puerto `Storage` sobre la API S3

- **Estado:** aceptada
- **Fecha:** 2026-10-23
- **Decide sobre:** SDD §2 (tabla de stack y nota de storage), §12

## Contexto

Necesitamos guardar fotos de locales, imágenes de servicios y, más adelante, el archivo PMTiles de los tiles del mapa. Los binarios no deberían pasar por la API: conviene que el navegador suba y baje directo contra el almacenamiento con URLs prefirmadas.

En desarrollo no hay credenciales de nube ni ganas de depender de la red para levantar el proyecto. En producción no queremos operar un servidor de objetos propio ni quedar atados a un proveedor.

## Decisión

1. **El código habla con un puerto `Storage` cuyo contrato es la API S3.** No hay SDK propietario ni cliente específico en el dominio: el servidor de objetos es un detalle de entorno.
2. **En desarrollo corre RustFS**, con la imagen pineada en `deploy/docker-compose.yml` (mismos puertos 9000/9001 que S3, un solo contenedor, licencia Apache-2.0). **Solo desarrollo**: no es una decisión de producción.
3. **En producción, un servicio gestionado:** Cloudflare R2, Backblaze B2 o AWS S3. Elegir uno es configuración de despliegue, no un cambio de código.
4. **Garage** queda como plan B si RustFS resulta inestable o incómodo para el equipo.
5. **Configuración por entorno:** endpoint, región, credenciales y **modo path-style** ajustables. R2 y B2 se usan en path-style; S3 permite virtual-host. El adaptador soporta ambos por configuración.
6. **El adaptador se prueba de verdad:** un test de integración (tag `integration`, Fase 2.4) sube un objeto y lo baja con URL prefirmada contra el servidor local. Un fake en memoria cubre el resto de los tests.

## Consecuencias

- Cambiar de proveedor de storage es cambiar variables de entorno; el dominio y los casos de uso no se tocan.
- La API nunca proxya binarios: entrega URLs prefirmadas con vencimiento corto.
- Los buckets son privados; el acceso se da por URL firmada. Las fotos de DNI de dueños van en un bucket separado con acceso restringido y se eliminan al resolver la verificación (SDD §7.1, Ley 25.326).
- El costo es que ningún test de integración del dominio corre sin un `Storage`: por eso el fake en memoria, y por eso el test de integración va con tag propio y no en el camino normal de `make test`.
- La versión exacta de la imagen vive en el compose, no en este documento: se actualiza ahí y se revisa en el PR.

## Alternativas descartadas

- **MinIO** (la elección inicial del SDD): se reemplazó por RustFS, que ocupa los mismos puertos, es un solo contenedor y tiene licencia Apache-2.0.
- **Garage como opción principal:** funciona, pero su licencia AGPL agrega preguntas que un proyecto con suscripciones no necesita responder hoy. Queda como plan B probado.
- **SeaweedFS:** cuatro contenedores para el mismo trabajo de desarrollo.
- **Un cliente propio por proveedor** (R2/S3/B2): tres adaptadores que hacen lo mismo y solo uno se usa por entorno.
- **Servir los binarios a través de la API Go:** pone el ancho de banda de las fotos en el proceso HTTP y en el mismo pool de conexiones que los datos.
