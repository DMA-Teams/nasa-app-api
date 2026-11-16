# nasa-app-api

API en Go que proyecta rotaciones de cultivos a 12 meses para cada municipio de Honduras, a partir
de condiciones climáticas y de suelo. Proyecto del reto **Field Shift** de NASA Space Apps 2026.

La recomendación usa solo clima y suelo. No considera precios, costos, plagas, mercado ni
rendimientos oficiales.

## Cómo funciona

- Un proceso ETL (`cmd/etl`) descarga los datos, arma un perfil climático y de suelo por municipio
  (299) y lo publica en Redis como un dataset versionado.
- La API (`cmd/api`) lee solo de Redis. Ningún pedido de usuario llama a servicios externos.
- El motor (`internal/agro`) simula el balance hídrico FAO-56 de cada rotación candidata con el
  clima histórico y el pronóstico estacional, y la puntúa. Las respuestas traen razones
  estructuradas; el texto lo arma el frontend.
- La latitud y la longitud del usuario se usan solo para encontrar el municipio. No se guardan ni se
  registran.

## Endpoints

| Método y ruta | Qué devuelve |
| ------------- | ------------ |
| `GET /v1/plan?lat=&lon=` | Plan del municipio que contiene el punto |
| `GET /v1/municipalities/{code}/plan` | Plan de un municipio |
| `GET /v1/departments/{code}/plan` | Plan agregado de un departamento |
| `GET /v1/municipalities`, `GET /v1/departments` | Listados |
| `GET /v1/crops` | Ficha de los 22 cultivos |
| `GET /livez`, `GET /readyz` | Salud |
| `GET /docs`, `GET /openapi.yaml` | Swagger UI y contrato OpenAPI 3.1 |

Parámetros opcionales del plan: `currentRotation`, `currentCrop`, `fieldFreeFrom`, `irrigation` y
`priority`. El detalle está en `api/openapi.yaml`.

La API es pública, sin login, con límite de pedidos por IP. Los errores siguen RFC 9457
(`application/problem+json`).

## Correr y probar

Requiere Go 1.27 y, para correr la API, Redis 8 (o Docker).

```bash
go test -race ./...
golangci-lint run ./...
govulncheck ./...
```

Entorno local con Docker, carga de datos y despliegue: `docs/operacion.md`.

## Documentación

| Documento | Contenido |
| --------- | --------- |
| `docs/plan-implementacion.md` | Alcance, base científica y decisiones |
| `docs/arquitectura.md` | Paquetes, flujos, datos en Redis, concurrencia y resiliencia |
| `docs/fuentes.md` | Origen de cada constante científica y agronómica |
| `docs/operacion.md` | Carga de datos, despliegue y verificación |
| `docs/calibracion.md` | Calibración del motor y decisiones de alcance |

## Fuentes de datos y atribución

| Fuente | Uso | Condiciones |
| ------ | --- | ----------- |
| [NASA POWER](https://power.larc.nasa.gov/) (NASA Langley Research Center) | Clima diario histórico | Datos públicos; citar a NASA POWER |
| [SoilGrids](https://soilgrids.org/) (ISRIC) | Propiedades del suelo | CC BY 4.0 |
| [geoBoundaries](https://www.geoboundaries.org/) | Límites de municipios y departamentos | Municipios CC BY 4.0 (ICF); departamentos ODbL (OpenStreetMap) |
| [Open-Meteo](https://open-meteo.com/) (ECMWF SEAS5, Copernicus DEM) | Pronóstico estacional y altitud | CC BY 4.0; la API gratuita es solo para uso no comercial |
| GPM IMERG (NASA GES DISC) | Solo calibración (`cmd/calibrate`) | Datos públicos; citar si se publican resultados |

Los archivos de `testdata/` son derivados de estas fuentes y quedan sujetos a las mismas
condiciones.

## Licencia

Este repositorio no tiene licencia de código abierto. El código se publica para consulta: todos los
derechos quedan reservados por sus autores, y copiarlo, modificarlo o redistribuirlo requiere
permiso. Los datos de terceros mantienen sus propias licencias (ver la tabla anterior).

Las skills de `.claude/skills/golang-*` son de terceros y tienen su propia licencia
(`.claude/skills/LICENSE-samber-cc-skills-golang`).
