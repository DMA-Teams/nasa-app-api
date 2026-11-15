---
name: go-resiliencia
description: "Resiliencia de la API en Go de este proyecto: timeouts del servidor HTTP, apagado ordenado, recuperación de panics, límite de pedidos por IP, descarte de carga, concurrencia acotada, singleflight, cliente de Redis con timeouts, health checks, y clientes HTTP del ETL con reintentos con backoff y jitter, Retry-After y versionado atómico de datos. Usar al escribir o revisar cmd/api, internal/httpapi, internal/store, internal/sources, cmd/etl, middlewares, configuración de Redis o de contenedores. Complementa golang-context, golang-concurrency y golang-design-patterns; ante conflicto, manda esta skill porque fija los valores del proyecto."
paths:
  - "**/*.go"
  - "deploy/**"
  - "docker-compose*.yml"
  - "Dockerfile"
---

# Resiliencia en Go para nasa-app-api

Esta skill fija **cómo** se protege la API y **con qué valores**. Las skills `golang-*` explican los
fundamentos; esta decide para este proyecto.

## Principios

1. **Todo tiene límite**: tiempo (timeout), tamaño (cuerpo, respuesta), concurrencia (semáforo,
   `errgroup.SetLimit`), memoria (`GOMEMLIMIT`, límite del contenedor) y pedidos (rate limit).
   Un recurso sin límite es una caída en espera.
2. **El `context.Context` manda**: toda función que hace I/O o cálculo largo recibe `ctx` como
   primer parámetro y lo respeta. Nunca se guarda `ctx` en un struct.
3. **Fallar rápido y degradar con honestidad**: si una dependencia no responde, se devuelve un
   error claro (503 con `Retry-After`), no se espera indefinidamente ni se inventan datos.
4. **Reintentar solo lo idempotente**, con backoff exponencial, jitter, tope de intentos y respeto
   de `Retry-After`. Nunca reintentar dentro del camino de un pedido de usuario.
5. **Cada goroutine tiene dueño, salida y manejo de panic**. Si no se puede decir cómo termina, no
   se lanza.
6. **No agregar patrones sin una dependencia que los justifique**. Hoy la API en ejecución solo
   depende de Redis local: no lleva circuit breaker (ver § Circuit breaker).

## Dónde está cada riesgo

| Proceso | Depende de | Riesgo principal | Defensa |
| ------- | ---------- | ---------------- | ------- |
| `cmd/api` (pedidos de usuarios) | Redis local | Redis lento o caído; picos de tráfico; pedidos departamentales pesados | Timeouts cortos, rate limit, descarte de carga, concurrencia acotada, singleflight |
| `cmd/etl` (precarga y pronóstico mensual) | NASA POWER, ISRIC WebDAV, Open-Meteo | APIs lentas, 429, 5xx, cortes a mitad de carga | Reintentos con backoff y jitter, limitador por fuente, carga versionada y atómica, reanudación |

## Servidor HTTP

Nunca usar `http.ListenAndServe` ni un `http.Server` sin timeouts.

| Campo | Valor inicial | Motivo |
| ----- | ------------- | ------ |
| `ReadHeaderTimeout` | 5 s | Corta clientes lentos al enviar cabeceras (Slowloris) |
| `ReadTimeout` | 10 s | Los cuerpos de este API son chicos |
| `WriteTimeout` | 30 s | Mayor que el timeout por pedido más alto |
| `IdleTimeout` | 60 s | Libera conexiones keep-alive ociosas |
| `MaxHeaderBytes` | 64 KiB | Cabeceras acotadas |

- Cuerpo de pedido acotado con `http.MaxBytesReader` (16 KiB alcanza para los pedidos de este API).
- Timeout **por pedido** con `context.WithTimeout` en un middleware: 5 s para municipio, 20 s para
  departamento. Al vencer, el handler devuelve 503 y deja de trabajar porque todo respeta `ctx`.
- Los valores salen de la configuración (variables de entorno) con estos valores por defecto.

### Orden de middlewares (de afuera hacia adentro)

1. Recuperación de panic → responde 500 en formato problem+json y registra el stack con `slog`.
2. ID de pedido (generado o leído de `X-Request-Id`).
3. Log de acceso (método, ruta, estado, duración, ID). **Nunca** la latitud ni la longitud del
   usuario: se registra el código de municipio.
4. Límite de pedidos por IP.
5. Descarte de carga (máximo de pedidos pesados en curso).
6. Timeout por pedido.
7. Handler.

## Apagado ordenado

1. `ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`.
2. Al recibir la señal: marcar el proceso como **no listo** (`/readyz` devuelve 503) para que el
   proxy deje de mandar tráfico.
3. `srv.Shutdown(ctxConTimeout)` con 15 s de plazo: deja terminar los pedidos en curso.
4. Esperar a las goroutines propias (`errgroup.Wait`).
5. Cerrar el cliente de Redis.
6. Salir con código distinto de 0 si algo falló, para que el orquestador lo vea.

`docker stop` envía SIGTERM y espera 10 s por defecto: configurar `stop_grace_period: 20s` en
compose para que el plazo de 15 s se cumpla.

## Panics

- `net/http` recupera los panics de los handlers, pero corta la conexión y los registra en el log
  estándar. El middleware de recuperación los convierte en una respuesta 500 limpia.
- **Un panic en una goroutine propia tira el proceso entero.** Toda goroutine lanzada por el código
  del proyecto recupera su propio panic y lo convierte en error, o corre dentro de un `errgroup`.
  Verificar en la versión de `golang.org/x/sync` usada cómo propaga `errgroup` los panics antes de
  confiar en eso.
- `panic` solo para invariantes rotas (bugs). Nunca para errores esperados.

## Límite de pedidos por IP

- `golang.org/x/time/rate`: un `rate.Limiter` por IP en un mapa protegido con `sync.Mutex`, con
  limpieza periódica de las IP inactivas (el mapa **no** crece sin límite). La goroutine de limpieza
  termina con el `ctx` del servidor.
- Valores iniciales: 2 pedidos por segundo con ráfaga de 10 por IP para municipio; 1 cada 10 s con
  ráfaga de 2 para departamento.
- La IP real sale de `X-Forwarded-For` **solo** si el pedido viene del proxy de confianza (Caddy en
  la misma red de Docker). Si no, de `RemoteAddr`.
- Al exceder: 429 con `Retry-After`.

## Descarte de carga y concurrencia acotada

- Los pedidos departamentales calculan hasta 23 municipios (Olancho). Un semáforo global limita los
  pedidos pesados en curso (valor inicial: `GOMAXPROCS`). Si está lleno, se responde **de inmediato**
  503 con `Retry-After: 5`, sin encolar.
- Dentro de un pedido departamental, los municipios se calculan con `errgroup.WithContext` y
  `SetLimit(runtime.GOMAXPROCS(0))`. El primer error cancela al resto.
- Nada de goroutines por elemento sin límite.

## Singleflight y caché de resultados

- La clave del resultado incluye todo lo que lo determina (municipio, fecha, riego, rotación actual,
  prioridad, versión de datos, de ficha y de pronóstico).
- Flujo: leer resultado en Redis → si no está, calcular dentro de `singleflight.Group.DoChan` con esa
  clave → guardar en Redis con TTL → responder.
- Con `DoChan`, el pedido que espera hace `select` entre el resultado y `ctx.Done()`: si su cliente
  se va, deja de esperar sin cancelar el cálculo compartido.
- El cálculo compartido corre con `context.WithoutCancel(ctx)` más su propio timeout, para que la
  cancelación de **un** cliente no rompa el resultado de los demás.
- Si falla la escritura en Redis, se registra y se responde igual: el cálculo ya está hecho.

## Cliente de Redis (go-redis v9)

| Opción | Valor inicial | Motivo |
| ------ | ------------- | ------ |
| `DialTimeout` | 2 s | Redis es local |
| `ReadTimeout` / `WriteTimeout` | 500 ms | Los blobs más grandes son ~600 KB en red local |
| `ContextTimeoutEnabled` | `true` | Respetar los deadlines del `ctx` de cada pedido |
| `PoolSize` | 10 × GOMAXPROCS (valor por defecto de go-redis) | Ajustar solo con medición |
| `PoolTimeout` | 1 s | No esperar indefinidamente por una conexión del pool |
| `MaxRetries` | 2 | Las lecturas son idempotentes |

- Al arrancar: `PING` con reintentos acotados (por ejemplo, 10 intentos en 20 s). Si no responde,
  salir con error: el contenedor se reinicia por su política.
- Al arrancar y periódicamente: leer el puntero de versión de datos (`dataset:current`). Si no
  existe, la API **no está lista**.
- Si Redis falla durante un pedido: 503 problem+json con `Retry-After`. Sin datos inventados.

## Circuit breaker

**No en el MVP.** La única dependencia en tiempo de ejecución es Redis en la misma máquina, con
timeouts cortos y `PoolTimeout`: eso ya corta la espera. Un breaker suma estado y configuración sin
beneficio medible. Se agrega (candidato: `github.com/sony/gobreaker/v2`) solo si aparece una
dependencia **remota** consultada en el camino del pedido, por ejemplo Supabase.

## Clientes HTTP del ETL

- Un `http.Client` por fuente, **nunca** `http.DefaultClient`. `Timeout` total por llamada (POWER:
  60 s, Open-Meteo: 30 s, ISRIC: el que necesite la descarga de rásteres) y `Transport` con
  `TLSHandshakeTimeout` y `ResponseHeaderTimeout`.
- `req = req.WithContext(ctx)` siempre. Cerrar el cuerpo con `defer resp.Body.Close()`, y leer con
  `io.LimitReader` (tope por fuente, por ejemplo 16 MiB) para no tragar respuestas gigantes.
- Reintentar **solo GET**, ante errores de red con timeout, 429 o 5xx. **No** ante 4xx distintos de
  429: son errores nuestros.
- Backoff exponencial con **jitter completo**: espera = aleatorio entre 0 y min(tope, base × 2^intento).
  Base 1 s, tope 60 s, máximo 6 intentos. Si hay `Retry-After`, se usa ese valor.
- La espera se hace con `select` sobre `ctx.Done()` y un `time.Timer`, nunca con `time.Sleep`.
- Un limitador `rate.Limiter` por fuente (POWER: 1 pedido por segundo; Open-Meteo: 2 por segundo)
  para no castigar servicios públicos compartidos.
- Implementación propia y chica en `internal/sources` (~60 líneas probadas). Alternativa
  evaluada: `github.com/failsafe-go/failsafe-go` (v0.9.x, anterior a 1.0). No se adopta mientras el
  código propio alcance.

## Carga de datos versionada y atómica

- El ETL escribe con un prefijo de versión nuevo (`v2026-11-14:...`), verifica conteos y, recién al
  final, cambia el puntero `dataset:current` con un solo `SET`. La API lee siempre a través del
  puntero: nunca ve una carga a medias.
- Cada unidad (celda, municipio) se escribe de forma idempotente: si el ETL se corta, se vuelve a
  correr y saltea lo que ya existe en esa versión.
- Las versiones viejas se borran después de un período de gracia, nunca antes de cambiar el puntero.

## Health checks

| Ruta | Responde 200 cuando | Uso |
| ---- | ------------------- | --- |
| `/livez` | El proceso responde | Reinicio del contenedor |
| `/readyz` | Redis responde **y** existe `dataset:current` **y** el proceso no está apagándose | Proxy y monitoreo |

Si el pronóstico tiene más de 45 días, `/readyz` sigue en 200, pero las respuestas del plan llevan la
advertencia `forecastStale`: es degradación, no indisponibilidad.

## Contenedor y runtime

- `restart: unless-stopped`, límite de memoria del contenedor y `GOMEMLIMIT` al ~85 % de ese límite,
  para que el GC trabaje antes de que el kernel mate el proceso.
- Desde Go 1.25, `GOMAXPROCS` respeta el límite de CPU del contenedor (cgroup). No fijarlo a mano.
- Imagen final mínima (distroless `static`, usuario sin privilegios).

## Cómo se prueba la resiliencia

- Los handlers se prueban con `httptest` y un store falso que devuelve errores, demoras y timeouts:
  verificar 503, 429, `Retry-After` y que el handler termina al cancelarse el `ctx`.
- Los reintentos se prueban con un `httptest.Server` que responde 429 o 500 N veces. Los tiempos se
  controlan con `testing/synctest` (estable desde Go 1.25) para no dormir de verdad en los tests.
- `go test -race ./...` siempre, y `go.uber.org/goleak` en los paquetes que lanzan goroutines.
- El apagado ordenado se prueba levantando el servidor, mandando un pedido lento y cancelando: el
  pedido termina y el servidor sale sin goroutines vivas.

## Lista de verificación de una revisión

- [ ] ¿Toda llamada de I/O recibe `ctx` y tiene timeout?
- [ ] ¿Cada goroutine tiene dueño, salida y manejo de panic?
- [ ] ¿Hay algún mapa, cola, slice o canal que pueda crecer sin límite?
- [ ] ¿Algún reintento sin backoff, sin jitter o sobre un método no idempotente?
- [ ] ¿Se registra la ubicación del usuario en algún log?
- [ ] ¿Algún error se traga, o se registra y además se devuelve?
- [ ] ¿Los valores de timeouts y límites salen de la configuración, con los valores de esta skill
      por defecto?

## Fuentes

- `net/http`, `http.Server`, `Shutdown`: https://pkg.go.dev/net/http
- `os/signal.NotifyContext`: https://pkg.go.dev/os/signal
- `golang.org/x/time/rate`: https://pkg.go.dev/golang.org/x/time/rate
- `golang.org/x/sync` (`errgroup`, `singleflight`, `semaphore`): https://pkg.go.dev/golang.org/x/sync
- go-redis v9, opciones del cliente: https://github.com/redis/go-redis/blob/v9.23.0/options.go
- Backoff con jitter (AWS Architecture Blog): https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
- `testing/synctest`: https://pkg.go.dev/testing/synctest
- failsafe-go (alternativa evaluada): https://failsafe-go.dev
- sony/gobreaker v2 (si se necesita un breaker): https://github.com/sony/gobreaker
