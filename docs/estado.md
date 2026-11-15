# Estado del proyecto y camino a producción

> Actualizado: 2026-11-15. Rama `feat/fase-0-api`, **sin commits** (los hace el usuario).

## 1. Qué está hecho

| Componente | Estado | Verificación |
| ---------- | ------ | ------------ |
| Núcleo científico (`internal/agro`) | Hecho | Ejemplos FAO-56 2, 8 y 18; aceptación con clima sintético |
| Motor de rotaciones | Hecho, **falta calibrar** | Ver `docs/calibracion.md` |
| Ficha de 22 cultivos | Hecha, muchos valores C | El validador exige fuente y confianza |
| API HTTP, Swagger (`/docs`, `/openapi.yaml`) | Hecha | Pruebas y prueba real en WSL |
| Resiliencia (timeouts, límite por IP, rechazo por carga, panics, apagado) | Hecha | Pruebas; el 429 se vio en la prueba real |
| Redis (binario, versiones, caché, vigilancia) | Hecho | Prueba de integración contra Redis real |
| ETL (carga versionada y reanudable, pronóstico mensual) | Hecho, probado en Lempira | Carga real: 28 perfiles en 35 s con suelo ráster |
| Entorno local WSL (`deploy/docker-compose.dev.yml`) | Hecho | Funcionando |
| Despliegue Lightsail (`deploy/docker-compose.yml`, Caddy, systemd) | Escrito, **no probado** | — |
| CI, monitoreo, backups | **No existe** | — |

Verificación al cierre: `go test -race ./...` en verde, `golangci-lint` sin hallazgos, `govulncheck`
sin vulnerabilidades.

Prueba real en WSL (Lempira): plan municipal en 32 ms sin caché y 1,5 ms con caché; plan
departamental de 28 municipios en 127 ms. Resultados en `data/smoke/` (no versionado).

## 2. Decisiones de esta sesión

- **Alcance:** solo clima y suelo; sin precios, costos, plagas ni mercado (`docs/calibracion.md` § 1).
- SoilGrids REST estuvo caído el 2026-11-15; la ruta confiable es el ráster por WebDAV
  (`deploy/soil/prepare.sh`).
- El usuario creó su cuenta de NASA Earthdata (token guardado fuera del repositorio).

## 3. Próxima sesión, en orden

1. **Calibración, etapa 1**: implementada y revisada por el usuario (fechas libres según las
   fórmulas; regla de cultivo de cosecha confirmada). Ver `docs/calibracion.md` § 0.
2. **Calibración, etapa 2**: IMERG, SMAP y NDVI con la cuenta de Earthdata.
3. Carga de todo el país en WSL: suelo ráster de Honduras completo, tiempo **no medido**; probar con
   una capa primero.

## 4. Pendiente para producción

### Datos

- Carga de los 18 departamentos.
- Revisar los 299 polígonos de geoBoundaries contra los 298 municipios oficiales.

### Infraestructura (Lightsail)

- Instancia de 4 GB. Firewall: 80 y 443 abiertos; 22 solo para las IP del equipo; nunca 6379 ni 6060.
- Dominio y DNS para Caddy (HTTPS automático).
- `deploy/.env` con una contraseña fuerte de Redis y los `CORS_ORIGINS` del frontend.
- Activar `deploy/systemd/nasa-forecast.timer`.
- Snapshot de la instancia después de cada carga.
- Límite de tamaño de los logs de Docker.
- Monitoreo externo de `/readyz`.
- Prueba de carga del plan departamental: la CPU de Lightsail funciona por créditos.

### Seguridad

- Revisión con la skill `golang-security` antes de salir.
- Decidir si `/docs` queda público en producción.
- Ajustar los límites por IP con tráfico real.

### Proceso

- CI con `go test -race`, `golangci-lint`, `govulncheck` y la integración con Redis.
- Antes del primer commit: `.mcp.json` tiene la ruta absoluta de `gopls` de la máquina del usuario.
- Comentarios godoc: el hook global del usuario los bloquea (`.claude/CLAUDE.md` § Comentarios).

### Contrato con el frontend

- Congelar el OpenAPI v1 (la prioridad `income` ya se quitó).
- Textos para cada código de razón.

### Atribución y licencias (mostrar en la app)

| Fuente | Condición |
| ------ | --------- |
| NASA POWER | Público, con cita |
| SoilGrids | CC BY 4.0 |
| geoBoundaries | Municipios CC BY 4.0; departamentos ODbL |
| Open-Meteo / ECMWF SEAS5 | Gratis solo para uso no comercial; citar a ECMWF |
| Copernicus DEM | Con cita |
| GPM IMERG, SMAP, MODIS | Solo para calibrar; citar si se publican resultados |

## 5. Gotchas del entorno local

- Docker corre **dentro de Ubuntu (WSL)**, no en Docker Desktop de Windows.
- Desde Git Bash, `wsl.exe` pierde las variables `$VAR` y Git Bash reescribe las rutas `/mnt/...`.
  Correr scripts con `MSYS_NO_PATHCONV=1 wsl.exe -d Ubuntu -- bash /mnt/d/.../script.sh`.
- La VM de WSL se reinició una vez y bajó todos los contenedores. El compose de desarrollo ya tiene
  `restart: unless-stopped`, y Redis guarda cada 5 minutos.
