# Arquitectura: nasa-app-api (Go)

> Estado: **implementada**. Se escribió como diseño el 2026-11-14, antes del código; el avance real
> está en `docs/estado.md`.
>
> Se apoya en `docs/plan-implementacion.md` (qué se construye y por qué) y en las skills de
> `.claude/skills/` (cómo se escribe). Este documento decide la estructura, los flujos y los valores
> concretos para cinco objetivos: código limpio, concurrencia, arquitectura, resiliencia y
> eficiencia.

---

## 1. Restricciones que dan forma a la arquitectura

| Restricción | Origen | Consecuencia |
| ----------- | ------ | ------------ |
| La unidad de análisis es el municipio (299 polígonos) o el departamento (18) | Plan § 0.2 | Los perfiles de sitio se precalculan; el pedido solo evalúa rotaciones |
| Ningún pedido de usuario llama a APIs externas | Plan § 4 | Las APIs externas viven solo en el ETL. La API en ejecución depende únicamente de Redis |
| Todo el dato de cálculo vive en Redis, en la misma instancia | Plan D4, D9 | Redis es local: latencia baja, pero comparte memoria y CPU con la API |
| API pública, sin login, sin estado | Plan Q1, Q2 | Escala horizontal trivial, protección por límite de pedidos por IP |
| El plan se recalcula en cada pedido con la fecha y el último pronóstico | Plan § 6.3 | Caché de resultados por clave completa, con versión del pronóstico |
| Lightsail con CPU por ráfagas (burstable) | Plan § 9.8 | Cálculo acotado por pedido, sin trabajos pesados permanentes |
| Sin agrónomo que valide | Plan D8 | Dominio puro, 100 % probado contra valores publicados |

---

## 2. Estilo: puertos y adaptadores livianos

### 2.1 Qué se eligió

El **dominio** (fórmulas, simulación, puntaje) está en el centro y no depende de nada. Los **casos
de uso** lo orquestan y declaran qué datos necesitan mediante interfaces chicas. Los **adaptadores**
(Redis, HTTP, fuentes externas) implementan esas interfaces. Los `main` arman todo a mano.

### 2.2 Por qué

- El dominio es la parte más riesgosa (ciencia sin revisión experta) y la más estable. Aislarlo
  permite probarlo sin Redis, sin red y sin reloj.
- Supabase va a entrar más adelante: hoy es un adaptador más que nadie implementa, sin tocar el
  dominio.
- Los mismos casos de uso sirven a la API y a los tests de aceptación agronómica.

### 2.3 Lo que se descartó

| Alternativa | Por qué no |
| ----------- | ---------- |
| Clean Architecture a la C# (capas Domain/Application/Infrastructure/Presentation, DTO por capa, mediador) | En Go produce paquetes con nombres genéricos, conversiones de DTO sin valor y ceremonia. La regla de dependencias se conserva sin las capas formales |
| DDD completo (agregados, repositorios por agregado, eventos) | No hay escrituras de negocio ni invariantes transaccionales: el sistema calcula y lee |
| Estructura plana (todo en `main` y un par de paquetes) | El dominio científico es grande y necesita aislamiento para probarlo |
| Framework web (Gin, Echo, Fiber) | `net/http` enruta por método y ruta desde Go 1.22, y los middlewares son funciones. Un framework suma dependencias y no resuelve nada que falte |

---

## 3. Paquetes y regla de dependencias

```
cmd/
  api/            main: configuración, armado de dependencias, servidor, apagado ordenado
  etl/            main: subcomandos power | soil | elevation | boundaries | forecast | profiles
internal/
  agro/           DOMINIO PURO: tipos con unidades, ET0, Kc, pedotransferencia, balance hídrico,
                  FAO-33, inicio de lluvias, canícula, grados-día, aptitud, RUSLE, nitrógeno,
                  generación y simulación de rotaciones, ponderación por pronóstico, puntaje
  catalog/        ficha de cultivos: JSON embebido (go:embed), esquema, validador, versión
  plan/           CASOS DE USO: plan municipal, plan departamental agrupado, perfil de sitio.
                  Declara las interfaces que necesita (ProfileReader, ForecastReader, ResultCache)
  geo/            límites administrativos embebidos, punto en polígono, celdas de POWER y SEAS5
  store/          ADAPTADOR Redis: claves, codificación binaria, versión de datos, caché
  sources/        ADAPTADORES de fuentes (solo los usa el ETL): POWER, Open-Meteo estacional y
                  elevación, lector de binarios de GDAL; cliente HTTP con reintentos
  httpapi/        ADAPTADOR HTTP: rutas, validación, problem+json, middlewares
  config/         lectura y validación de variables de entorno
testdata/         respuestas grabadas de POWER, SEAS5 y SoilGrids; casos de aceptación
deploy/           Dockerfile, docker-compose.yml, Caddyfile, redis.conf
```

### Regla de dependencias (la verifica `depguard` en `golangci-lint`)

```
cmd/api ──► httpapi ──► plan ──► agro
   │           │          │        ▲
   │           │          └──► catalog
   │           └──► geo
   └──► store ─────────────────────┘  (store implementa interfaces de plan; usa tipos de agro)

cmd/etl ──► sources, store, geo, agro, catalog
```

- `agro` no importa ningún otro paquete interno ni nada de I/O.
- `plan` no importa `store`, `httpapi` ni `sources`: solo conoce sus propias interfaces.
- `httpapi` no importa `store`: recibe los casos de uso ya armados.
- Ningún paquete importa `cmd/`.

### Nombres

Paquetes en singular, cortos y por **lo que proveen**, no por capa (`agro`, no `domain`; `store`,
no `repository`). Sin `util`, `common` ni `helpers` (`golang-naming`).

---

## 4. Flujos principales

### 4.1 Plan por ubicación o municipio (`GET /v1/municipalities/{code}/plan`)

```
cliente
  │ GET /v1/plan?lat=..&lon=..  ó  /v1/municipalities/{code}/plan?irrigation=..&priority=..&current=..
  ▼
httpapi: límite por IP → validación → (lat, lon) ─► geo.MunicipalityAt ─► código (lat/lon no se guardan)
  ▼
plan.MunicipalPlan(ctx, código, fecha de hoy, entradas)
  │ 1. clave de resultado = hash(código, fecha, entradas, versión de datos, ficha y pronóstico)
  │ 2. ResultCache.Get(clave) ──► acierto: responder
  │ 3. singleflight.DoChan(clave):
  │      ProfileReader.Profile(código)        perfil precalculado (clima por quincena, suelo, altitud)
  │      ClimateReader.DailySeries(celdas)    serie diaria de 30 años de las celdas del municipio
  │      ForecastReader.Weights(celdas, mes)  pesos de los años para los próximos 7 meses
  │      agro.EvaluateRotations(...)          generación, simulación, puntaje (puro, sin I/O)
  │      ResultCache.Set(clave, resultado, TTL 35 días)
  ▼
httpapi: JSON con trimestres, semestres, temporadas, razones estructuradas, versiones, advertencias
```

La ubicación se resuelve a municipio en el borde. Los casos de uso nunca reciben coordenadas.

### 4.2 Plan departamental agrupado (`GET /v1/departments/{code}/plan`)

1. El semáforo de pedidos pesados admite el pedido o responde 503 de inmediato.
2. `errgroup.WithContext` + `SetLimit(GOMAXPROCS)`: un `MunicipalPlan` por municipio. Cada uno
   reutiliza la caché y el singleflight del flujo 4.1.
3. Agrupar por la secuencia recomendada de cultivos por trimestre. Por grupo: municipios y rango de
   las métricas.
4. El primer error cancela al resto y el pedido responde con problem+json.

### 4.3 ETL inicial (una vez) y mensual

| Subcomando | Frecuencia | Qué hace |
| ---------- | ---------- | -------- |
| `etl boundaries` | Una vez | Convierte geoBoundaries ADM1/ADM2 a un archivo compacto embebido en `geo` |
| `etl power` | Una vez + cada enero | 88 celdas × 30 años; control de calidad; ET0 diaria; escribe en Redis |
| `etl soil` | Una vez | Lee los binarios que dejó GDAL (Docker), agrega por municipio, escribe en Redis |
| `etl elevation` | Una vez | ~4.500 puntos de Copernicus DEM; mediana, mínima y máxima por municipio |
| `etl profiles` | Tras cualquiera de los anteriores | Precalcula los 299 perfiles municipales |
| `etl forecast` | Mensual (día 6) | SEAS5 por celda → pesos por tercil por mes; nueva versión de pronóstico |

Toda carga escribe con un prefijo de versión nuevo y cambia el puntero al final
(`go-resiliencia` § Carga versionada). El mensual se dispara con un temporizador de systemd o con
cron en la instancia, ejecutando el contenedor del ETL.

---

## 5. Datos en Redis

### 5.1 Claves

| Clave | Contenido | Escribe | TTL |
| ----- | --------- | ------- | --- |
| `dataset:current` | Versión vigente de datos estáticos, por ejemplo `d2026-11-14` | ETL | No |
| `forecast:current` | Versión vigente del pronóstico, por ejemplo `f2026-11` | ETL mensual | No |
| `{d}:climate:{celda}` | Serie diaria de 30 años, binaria | ETL | No |
| `{d}:climate:{celda}:qa` | Meses excluidos por el control de calidad | ETL | No |
| `{d}:profile:{municipio}` | Perfil precalculado: clima por quincena, suelo, altitud, celdas y pesos de área | ETL | No |
| `{f}:forecast:{celda}` | Probabilidades por tercil para los 7 meses | ETL mensual | 120 días |
| `result:{hash}` | Resultado de una evaluación | API | 35 días |

### 5.2 Codificación

- **Series climáticas**: binario little-endian. Cabecera fija (versión de formato, fecha inicial,
  cantidad de días, cantidad de parámetros, orden de parámetros) y luego un bloque `float32`
  contiguo por parámetro. Para decodificar basta con `encoding/binary` y conversión de bits a
  float32, sin reflexión: una celda (~570 KB) se lee y decodifica en milisegundos.
- **Perfiles y resultados**: JSON (chicos, legibles al depurar). Si el profiling muestra que JSON
  pesa, se cambia el formato detrás del adaptador sin tocar el dominio.
- Cada formato binario lleva su número de versión: un cambio de formato es una versión nueva de
  datos, nunca una sobrescritura.

### 5.3 Configuración de Redis

- `bind` a la red interna de Docker; `protected-mode yes`; usuario ACL con contraseña para la API
  (solo lectura sobre los datos y escritura sobre `result:*`) y otro para el ETL.
- `maxmemory 1gb`, `maxmemory-policy noeviction`. El dato estático jamás se desaloja; si la memoria
  se llena, falla la escritura de resultados y la API sigue respondiendo.
- Persistencia RDB, con `save` después de cada carga del ETL. El dato es reconstruible con el ETL.
- Memoria esperada: ~100 MB (plan § 9.2, v4). Se mide después de la primera carga.

---

## 6. Concurrencia

### 6.1 Modelo

- **Un pedido = una goroutine** (la que crea `net/http`). Dentro del pedido municipal, el cálculo es
  secuencial: simular 30 años de pocas rotaciones no justifica goroutines extra.
- **Paralelismo solo donde hay trabajo independiente y medido:** el pedido departamental (§ 4.2) y
  el ETL (celdas y municipios en paralelo, con `errgroup.SetLimit`).
- **Sin estado mutable global.** Lo compartido es inmutable (ficha de cultivos embebida, límites
  administrativos) o está protegido y acotado (limitadores por IP, singleflight).

### 6.2 Reglas (de `golang-concurrency` y `golang-context`)

- Toda goroutine tiene dueño, salida por `ctx` y manejo de panic.
- `ctx` como primer parámetro de toda función que hace I/O o cálculo largo. Los ciclos largos del
  dominio revisan `ctx.Err()` entre rotaciones, no dentro del ciclo diario (sería caro).
- Canales con dirección y cerrados solo por quien envía; sin buffers sin justificación medida.
- `-race` y `goleak` en los tests.

### 6.3 Por qué no un caché en memoria del dataset (todavía)

Cargar los ~100 MB de Redis en memoria del proceso y cambiarlos con `atomic.Pointer` al cambiar la
versión evitaría las lecturas a Redis. Es más rápido y sobreviviría a una caída breve de Redis, pero
duplica la memoria, agrega un ciclo de recarga y contradice la decisión de leer de Redis. Se adopta
**solo si los benchmarks muestran que la lectura de Redis domina la latencia**.

---

## 7. Resiliencia (resumen; detalle en la skill `go-resiliencia`)

| Mecanismo | Valor inicial |
| --------- | ------------- |
| Timeouts del servidor | ReadHeader 5 s, Read 10 s, Write 30 s, Idle 60 s |
| Timeout por pedido | Municipio 5 s, departamento 20 s |
| Cuerpo máximo | 16 KiB |
| Límite por IP | Municipio 2/s, ráfaga 10; departamento 1 cada 10 s, ráfaga 2 |
| Pedidos pesados simultáneos | `GOMAXPROCS`; excedente → 503 inmediato |
| Redis | Dial 2 s, lectura y escritura 500 ms, `ContextTimeoutEnabled`, `PoolTimeout` 1 s |
| Apagado | `/readyz` → 503, `Shutdown` con 15 s, `stop_grace_period` 20 s |
| Reintentos del ETL | Solo GET; 429, 5xx y timeouts; backoff exponencial con jitter completo, base 1 s, tope 60 s, 6 intentos; respeta `Retry-After` |
| Memoria | Límite del contenedor + `GOMEMLIMIT` al 85 % |

---

## 8. Eficiencia

### 8.1 Dónde está el costo

| Etapa | Costo esperado | Estrategia |
| ----- | -------------- | ---------- |
| Lectura de series de Redis | ~570 KB por celda; 1 a 3 celdas por municipio | Binario sin reflexión; medir |
| Simulación | Rotaciones × 30 años × 365 días | Ciclo sin asignaciones; slices preasignados; filtro de aptitud **antes** de simular |
| Plan departamental | Hasta 23 municipios | Paralelo acotado; caché y singleflight por municipio |
| Serialización de la respuesta | Decenas de KB | `encoding/json`; medir antes de cambiar |

### 8.2 Precalcular en el ETL lo que no depende del pedido

ET0 diaria, resúmenes por quincena, inicio de lluvias y canícula por año, agregados de suelo, altitud
y pesos de área por celda. El pedido solo hace lo que depende de la fecha, del pronóstico y de las
entradas del usuario.

### 8.3 Metas a validar con benchmarks (no son mediciones)

| Operación | Meta p95 |
| --------- | -------- |
| Plan municipal con caché | < 20 ms |
| Plan municipal sin caché | < 500 ms |
| Plan departamental de Olancho sin caché | < 3 s |

Se miden con `b.Loop()`, `benchstat` y `pprof` (`golang-benchmark`). Si no se cumplen, se perfila
antes de cambiar nada (`golang-performance`).

### 8.4 Diagnóstico en producción

`net/http/pprof` en un puerto separado, escuchando **solo en localhost** (se accede por túnel SSH).
Nunca expuesto por el proxy.

---

## 9. Contrato HTTP

### 9.1 Rutas (borrador)

| Método y ruta | Descripción |
| ------------- | ----------- |
| `GET /v1/plan?lat=&lon=&…` | Resuelve la ubicación a municipio y devuelve su plan |
| `GET /v1/municipalities` | Lista de municipios (código, nombre, departamento) |
| `GET /v1/municipalities/{code}/plan?…` | Plan de un municipio |
| `GET /v1/departments` | Lista de departamentos |
| `GET /v1/departments/{code}/plan?…` | Plan departamental agrupado por recomendación |
| `GET /v1/crops` | Ficha de cultivos con fuentes y confianza |
| `GET /livez`, `GET /readyz` | Salud |

Los parámetros comunes son `irrigation` (bool), `priority` (`balanced` \| `water` \| `soil`),
`currentCrop` y `currentRotation`. Todas las rutas son GET, idempotentes y cacheables. La fecha del
plan es la del servidor en hora de Honduras (`America/Tegucigalpa`), no la que manda el cliente.

### 9.2 Errores

- `application/problem+json` (RFC 9457) con `type`, `title`, `status`, `detail` y, cuando aplica,
  `errors` por campo.
- El dominio y los casos de uso devuelven errores centinela (`ErrMunicipalityNotFound`,
  `ErrOutsideHonduras`, `ErrDataUnavailable`). Solo `httpapi` los traduce a códigos HTTP
  (404, 422, 503). Los detalles técnicos se registran; al cliente no se le exponen.
- Regla de manejo único: un error se registra **o** se devuelve, nunca las dos cosas
  (`golang-error-handling`).

### 9.3 CORS

Lista explícita de orígenes del frontend, configurable. Sin `*` en producción.

---

## 10. Observabilidad

- `log/slog` con salida JSON a stdout. Atributos estables: `request_id`, `route`, `status`,
  `duration_ms`, `municipality`, `dataset_version`, `forecast_version`.
- **Nunca** coordenadas del usuario, ni en logs ni en métricas.
- Métricas mínimas con `expvar` (pedidos, errores por código, aciertos de caché, rechazos por límite)
  en el puerto de diagnóstico local. Prometheus y OpenTelemetry quedan para cuando haya dónde
  enviarlos (`golang-observability`).

---

## 11. Seguridad

- Validación en el borde: rangos de lat y lon, códigos de municipio contra la lista embebida, enums
  cerrados, tamaño del cuerpo.
- Redis no expuesto (`go-resiliencia`); secretos solo en variables de entorno.
- `govulncheck` y `gosec` (vía `golangci-lint`) en cada verificación. Revisión con
  `golang-security` antes del despliegue.
- Imagen distroless sin shell, usuario sin privilegios, sistema de archivos de solo lectura.

---

## 12. Pruebas

| Nivel | Qué | Cómo |
| ----- | --- | ---- |
| Dominio | Cada fórmula contra su ejemplo publicado; invariantes; casos límite | Tablas, `testing.F` para propiedades, sin I/O |
| Aceptación agronómica | Casos de plan § 5.6 con datos grabados | `testdata/`, entrada completa → salida esperada |
| Casos de uso | Plan municipal y departamental con fakes de los puertos | Fakes escritos a mano (sin librerías de mocks) |
| Adaptadores HTTP | Rutas, validación, problem+json, límites, timeouts | `httptest` |
| Adaptador Redis | Codificación y versionado contra un Redis real | Etiqueta de build `integration`, Redis en Docker |
| Clientes del ETL | Reintentos, `Retry-After`, límites de tamaño | `httptest.Server` + `testing/synctest` |
| Concurrencia | Carreras y fugas | `-race` siempre; `goleak` en paquetes con goroutines |
| Rendimiento | Simulación y plan completo | Benchmarks con `b.Loop()` |

Comando de verificación: `go test -race ./...`. Integración: `go test -race -tags integration ./...`.

---

## 13. Herramientas y configuración del repositorio

- `.golangci.yml`: parte del archivo de `golang-lint`, más `depguard` con la regla de dependencias
  de § 3. Sin `testifylint` (no se usa testify), y con `revive.exported` y `godot` según se decida
  sobre los comentarios (§ 16).
- `gopls` como servidor MCP (`.mcp.json`) para navegación y diagnósticos.
- Hook de `gofmt` tras cada edición de un `.go` (`.claude/hooks/gofmt-go.js`; ver § 16, falta
  registrarlo).
- Herramientas como dependencias de herramienta en `go.mod` (`go get -tool`, Go 1.24+) cuando sean
  módulos Go (`govulncheck`). `golangci-lint` se instala como binario, como recomienda su proyecto.

---

## 14. Configuración (variables de entorno)

| Variable | Por defecto | Uso |
| -------- | ----------- | --- |
| `HTTP_ADDR` | `:8080` | Dirección del servidor |
| `DIAG_ADDR` | `127.0.0.1:6060` | pprof y expvar, solo local |
| `REDIS_ADDR` | `redis:6379` | Redis |
| `REDIS_USERNAME`, `REDIS_PASSWORD` | sin valor; obligatorias | Credenciales ACL |
| `CORS_ORIGINS` | sin valor | Orígenes permitidos, separados por coma |
| `TRUSTED_PROXY_CIDRS` | red interna de Docker | De dónde se acepta `X-Forwarded-For` |
| `RATE_*`, `TIMEOUT_*`, `HEAVY_MAX_INFLIGHT` | valores de § 7 | Ajustes de resiliencia |
| `GOMEMLIMIT` | 85 % del límite del contenedor | Runtime de Go |

`config` valida todo al arrancar. Una configuración inválida impide arrancar, con un mensaje claro.

---

## 15. Despliegue en Lightsail

```
Internet ──443──► Caddy (TLS automático) ──► api:8080 ──► redis:6379
                                               │
                                     127.0.0.1:6060 (pprof/expvar, túnel SSH)
etl (contenedor a demanda; mensual por temporizador) ──► redis
```

- `Dockerfile` en dos etapas: `golang:1.27` para compilar (`CGO_ENABLED=0`, `-trimpath`) y
  la imagen distroless `static` más reciente con la etiqueta `nonroot` para ejecutar. La versión
  exacta de Debian de esa imagen se fija al escribir el Dockerfile.
- `docker-compose.yml`: `api`, `redis`, `caddy` y el perfil `etl`. `restart: unless-stopped`,
  límites de memoria, `stop_grace_period: 20s`, healthchecks contra `/readyz` y `redis-cli ping`.
- Firewall de Lightsail: 80 y 443 abiertos; 22 restringido a las IP del equipo; nada más.
- El ETL de suelo con GDAL corre en la máquina de desarrollo. A Lightsail solo llegan los agregados.

---

## 16. Dependencias permitidas

| Módulo | Versión al 2026-11-14 | Para qué |
| ------ | --------------------- | -------- |
| Librería estándar de Go 1.27 | 1.27.4 | Casi todo |
| `github.com/redis/go-redis/v9` | v9.23.0 | Cliente de Redis |
| `golang.org/x/sync` | v0.25.0 | `errgroup`, `singleflight`, `semaphore` |
| `golang.org/x/time` | v0.16.0 | `rate` |
| `github.com/paulmach/orb` | v0.14.0 | Geometría: punto en polígono, GeoJSON |
| `github.com/google/go-cmp` | última | Comparación en tests |
| `go.uber.org/goleak` | v1.3.0 | Fugas de goroutines en tests |

Evaluadas y no adoptadas por ahora: `failsafe-go` (v0.9.x, anterior a 1.0; el código propio de
reintentos alcanza), `sony/gobreaker/v2` (no hay dependencia remota en el camino del pedido),
frameworks web y librerías de DI.

---

## 17. Decisiones abiertas

1. ~~Comentarios de documentación~~ **Decidido (2026-11-15):** solo godoc en identificadores
   exportados y paquetes; `revive.exported` y `godot` activos.
2. ~~Ruta del módulo~~ **Decidido (2026-11-15):** `github.com/dma-teams/nasa-app-api`.
3. **Hook y permisos del proyecto**: `.claude/settings.json` no se pudo crear desde la sesión (la
   escritura de configuración de permisos y hooks requiere aprobación del usuario). Contenido
   propuesto en la respuesta de la sesión del 2026-11-14.
4. **Instalación de Go y herramientas** en la máquina de desarrollo (`.claude/CLAUDE.md`
   § Herramientas).
5. **Caché en memoria del dataset** (§ 6.3): solo si los benchmarks lo justifican.

---

## 18. Desviaciones de la implementación (2026-11-15)

| Diseño | Implementación | Motivo |
| ------ | -------------- | ------ |
| Límites administrativos embebidos con `go:embed` | Se guardan en Redis por versión de datos y la API los carga al cambiar `dataset:current` | El hook del usuario bloquea las directivas `//go:`; además así los límites quedan versionados con el resto de los datos |
| Ficha en JSON embebido | Constante Go generada por `tools/gencatalog/gen.js` | Mismo motivo |
| Prueba de integración con etiqueta `integration` | Se activa con la variable `REDIS_TEST_ADDR` | Mismo motivo |
| Comentarios godoc | Sin comentarios; constantes documentadas en `docs/fuentes.md` | Hook del usuario |
| Kc ini calculado con FAO-56 cap. 6 | Kc ini fijo de la ficha (valores típicos, confianza B) | Simplificación; queda para una versión posterior |
| Raíz que crece | Raíz a profundidad máxima desde la siembra | Simplificación; FAO-56 recomienda el valor alto para modelar secano |
| `cmd/etl` con toda la lógica | Lógica en `internal/etl`, `main` solo arma dependencias | Permite probar el ETL completo con fuentes simuladas |
| Usuarios ACL de Redis separados para API y ETL | Una contraseña (`requirepass`) | Simplificación del MVP; Redis no se expone fuera de la red de Docker |
| GeoJSON con `orb/geojson` | Lector propio y mínimo en `internal/geo/encode.go` | `orb/geojson` arrastra el driver de MongoDB como dependencia |
| Suelo ráster con Q0.05 y Q0.95 de SoilGrids | Ráster: solo la capa `mean`, y P05 y P95 describen la variación espacial dentro del municipio. REST: cuantiles del modelo en un punto | Reduce la descarga de 96 a 32 capas |
| Siembra en cualquier quincena | En secano, se exige humedad en la quincena de siembra (mediana de lluvia ≥ 0,5 × ET0) | Sin esta regla el motor sembraba a fines de abril, antes de las lluvias |
