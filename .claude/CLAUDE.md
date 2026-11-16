# nasa-app-api

API en Go que proyecta rotaciones de cultivos por municipio de Honduras a 12 meses, con datos de NASA
POWER, SoilGrids y el pronóstico estacional ECMWF SEAS5. Reto "Field Shift" de NASA Space Apps 2026.

## Leer primero, en este orden

1. `docs/plan-implementacion.md`: alcance, base científica y decisiones. Las secciones § 0.1, § 0.2
   y § 0.3 mandan sobre el resto del documento.
2. `docs/arquitectura.md`: paquetes, flujos, datos en Redis, concurrencia, resiliencia, eficiencia,
   pruebas y despliegue.
3. Las skills de `.claude/skills/` que apliquen a la tarea (ver § Skills).

4. `docs/fuentes.md` (origen de constantes y supuestos) y `docs/operacion.md` (carga y despliegue).
5. **`docs/estado.md` y `docs/calibracion.md`: dónde quedamos y qué sigue.** Empezar por acá en
   cada sesión nueva.

**Alcance (decisión del usuario, 2026-11-15):** la app recomienda solo con condiciones climáticas y
de suelo. Precios, costos, plagas, mercado y rendimientos oficiales están fuera del alcance: no
proponerlos.

**Credenciales:** el token de NASA Earthdata del usuario va en la variable de entorno
`EARTHDATA_TOKEN`. Nunca pedirlo en el chat ni escribirlo en archivos del repositorio.

La implementación completa se autorizó el 2026-11-15 y está en la rama `feat/fase-0-api`.

## Hechos del proyecto

- **Lenguaje:** Go 1.27.x (estable al 2026-11-14: go1.27.4).
- **Módulo:** `github.com/dma-teams/nasa-app-api` (repositorio `DMA-Teams/nasa-app-api`, en
  minúsculas por convención de Go). Confirmado por el usuario el 2026-11-15.
- **Arquitectura:** puertos y adaptadores livianos. El dominio (`internal/agro`) es puro y no conoce
  Redis ni HTTP. Las interfaces se declaran en el paquete que las consume. Detalle en
  `docs/arquitectura.md`.
- **Inyección de dependencias:** manual, por constructores, armada en `cmd/api/main.go` y
  `cmd/etl/main.go`. Sin librerías de DI.
- **Datos:** Redis en la misma instancia de Lightsail. Sin Supabase en el MVP.
- **API:** pública, sin login, con límite de pedidos por IP. Errores en formato RFC 9457
  (`application/problem+json`). Explicaciones como razones estructuradas: el texto lo arma el
  frontend.
- **Privacidad:** la latitud y la longitud del usuario se usan solo para encontrar el municipio. No
  se guardan ni se registran en logs.

## Skills

- Toda tarea de Go: cargar `golang-how-to`, que enruta al resto de las `golang-*`.
- Servidor HTTP, middlewares, Redis, clientes del ETL, Docker o configuración del runtime: cargar
  además `go-resiliencia`.
- Cualquier cosa en `internal/agro`, `internal/catalog`, la ficha de cultivos o sus tests: cargar
  además `agro-calculos`.
- Origen y cambios de las skills de terceros: `.claude/skills/VENDOR.md`.

## Precedencia sobre las skills

Las skills `golang-*` son de un tercero y traen valores por defecto genéricos. En este proyecto
mandan estas reglas:

1. **No preguntar de nuevo** por la arquitectura ni por el enfoque de inyección de dependencias:
   ya están decididos (arriba).
2. **No agregar librerías de `samber/*`** (`oops`, `lo`, `do`, `slog-*`, etc.) ni frameworks web. La
   librería estándar primero. Dependencias permitidas: las de `docs/arquitectura.md` § Dependencias.
   Cualquier otra se propone con su justificación antes de agregarla.
3. **Tests con la librería estándar** (`testing`, `httptest`, `testing/synctest`). Para comparar
   estructuras, `github.com/google/go-cmp`. Sin testify.
4. **Sin sub-agentes en paralelo** salvo que el usuario lo pida, aunque una skill describa un
   "Orchestration mode" o un "Audit mode" con varios sub-agentes.
5. **Sin circuit breaker** mientras la única dependencia en tiempo de ejecución sea Redis local
   (`go-resiliencia` § Circuit breaker).
6. Ante conflicto entre `go-resiliencia` o `agro-calculos` y una skill `golang-*`, mandan las del
   proyecto.

## Verificación

- La verificación de un cambio son los tests: `go test -race ./...`.
- Además: `golangci-lint run` (configuración en `.golangci.yml`, basada en la de `golang-lint`) y
  `govulncheck ./...`.
- Un `go build` solo prueba que compila. No cuenta como verificación.

## Comentarios en el código

El usuario aprobó el 2026-11-15 los comentarios godoc, pero **su hook global los bloquea** (y también
las directivas `//go:embed` y `//go:build`, porque empiezan con `//`). El hook solo los permite si se
piden en el mismo mensaje. Estado actual:

- **Ningún comentario en el código.** `revive.exported`, `revive.package-comments` y `godot` están
  desactivados en `.golangci.yml`.
- Sin `go:embed`: la ficha de cultivos es una constante Go generada (`internal/catalog/crops_data.go`).
- Sin etiquetas de build: la prueba de integración de Redis se activa con `REDIS_TEST_ADDR`.
- El origen de cada constante científica va en `docs/fuentes.md`.

Si el usuario agrega una excepción para godoc en su hook, reactivar esas reglas y documentar.

## Git

El agente no hace `commit`, `push`, `merge` ni `rebase`. Ramas y mensajes según la convención del
usuario (conventional commits).

## Herramientas

Estado al 2026-11-15 en la máquina de desarrollo: Go 1.27.4 (scoop), `gopls` v0.24.0,
`golangci-lint` y `govulncheck` instalados. El servidor MCP `gopls` de `.mcp.json` y el hook
`.claude/hooks/gofmt-go.js` funcionan.

Instalación en una máquina nueva:

```bash
scoop install go
go install golang.org/x/tools/gopls@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
scoop install golangci-lint
```

## Repositorio público

El repositorio se publica sin licencia (decisión del usuario, 2026-11-15): el código queda visible,
pero no se otorga permiso de reutilización. No agregar un archivo `LICENSE` sin pedirlo.

- `docs/servidor.md` (IP, nombre de la instancia, reglas de SSH) está en `.gitignore` y no se sube.
  Ningún archivo versionado debe tener la IP, el dominio de producción ni datos de la instancia.
- Los endpoints públicos no devuelven errores internos (`err.Error()`): el detalle va al log.
- La atribución de las fuentes de datos está en `README.md`.
