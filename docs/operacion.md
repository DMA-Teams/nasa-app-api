# Operación: cargar datos, desplegar y verificar

## Verificar el código

```bash
go test -race ./...
golangci-lint run ./...
"$(go env GOPATH)/bin/govulncheck" ./...
```

Prueba de integración contra un Redis real (opcional):

```bash
docker run --rm -d -p 6379:6379 --name redis-test redis:8.2-alpine
REDIS_TEST_ADDR=localhost:6379 go test ./internal/store/ -run Integration -v
docker stop redis-test
```

## Entorno local en WSL

Requiere Docker dentro de la distribución de WSL (verificado en Ubuntu con Docker 29.6 y Compose v5.2).
Todo se corre desde WSL, sobre la ruta del repositorio (en Windows, `/mnt/<unidad>/...`).

```bash
cd <ruta-del-repositorio>/deploy
docker compose -f docker-compose.dev.yml up -d --build redis api
curl http://127.0.0.1:8080/readyz
```

- Redis queda en `127.0.0.1:6379` con la contraseña `dev-only-password` (cambiarla con
  `REDIS_PASSWORD`). Solo escucha en localhost.
- La API queda en `127.0.0.1:8080` y el diagnóstico en `127.0.0.1:6060`. Hasta cargar datos,
  `/readyz` responde 503 `not-ready`: es lo esperado.
- Carga rápida de un solo departamento con las APIs reales:

  ```bash
  docker compose -f docker-compose.dev.yml --profile etl run --rm etl build -department=Lempira -version=d-dev-lempira
  docker compose -f docker-compose.dev.yml --profile etl run --rm etl forecast -department=Lempira
  ```

  Con `-department` solo se cargan los perfiles de ese departamento. La lista de municipios del país
  queda completa, pero el plan de un municipio de otro departamento responde 503
  `data-unavailable`.
- Prueba de integración desde Windows o WSL, contra ese Redis:

  ```bash
  REDIS_TEST_ADDR=localhost:6379 REDIS_TEST_PASSWORD=dev-only-password go test ./internal/store/ -run Integration -v
  ```

- Apagar: `docker compose -f docker-compose.dev.yml down`. Para borrar también los datos, agregar `-v`.

## Variables de entorno

| Variable | Obligatoria | Por defecto | Uso |
| -------- | ----------- | ----------- | --- |
| `REDIS_PASSWORD` | Sí, salvo `APP_ENV=development` | — | Contraseña de Redis |
| `REDIS_ADDR` | No | `redis:6379` | Dirección de Redis |
| `API_DOMAIN` | Sí (Caddy) | — | Dominio público con HTTPS automático |
| `CORS_ORIGINS` | No | vacío | Orígenes del frontend, separados por coma |
| `TIMEOUT_MUNICIPAL`, `TIMEOUT_DEPARTMENT`, `TIMEOUT_COMPUTE` | No | 5s, 20s, 10s | Timeouts |
| `RATE_MUNICIPAL_PER_SECOND`, `RATE_MUNICIPAL_BURST` | No | 2, 10 | Límite por IP |
| `RATE_DEPARTMENT_PER_SECOND`, `RATE_DEPARTMENT_BURST` | No | 0.1, 2 | Límite por IP |
| `HEAVY_MAX_INFLIGHT` | No | GOMAXPROCS | Pedidos departamentales simultáneos |
| `TRUSTED_PROXY_CIDRS` | No | redes privadas de Docker | De dónde se acepta `X-Forwarded-For` |
| `TIME_ZONE` | No | `America/Tegucigalpa` | Fecha de inicio de los planes |

Crear `deploy/.env` con esas variables. **No** se versiona (`.gitignore`).

## Carga inicial de datos (una vez)

1. Levantar Redis: `docker compose -f deploy/docker-compose.yml up -d redis`.
2. Suelo, una de dos:
   - **REST** (simple): no requiere preparación. Hace 1 llamada a SoilGrids por municipio con el
     límite de uso justo (5 por minuto): ~60 minutos para 299 municipios.
   - **Ráster a 250 m** (recomendado): en la máquina de desarrollo,
     `bash deploy/soil/prepare.sh data/soil`. Descarga y reproyecta 32 capas de SoilGrids con GDAL
     en Docker. Tiempo y volumen **sin medir**: probar primero con una sola capa.
3. Cargar el resto:

   ```bash
   docker compose -f deploy/docker-compose.yml --profile etl run --rm etl build -soil=rest
   docker compose -f deploy/docker-compose.yml --profile etl run --rm etl build -soil=raster -soil-dir=/data/soil
   ```

   El build descarga límites, 30 años de POWER por celda (1 pedido por segundo), altitud y suelo,
   arma los 299 perfiles y **recién al final** publica `dataset:current`. Si se corta, volver a
   correr el mismo comando con el mismo `-version`: saltea lo que ya está cargado.
4. Pronóstico: `docker compose ... --profile etl run --rm etl forecast`.

## Despliegue en Lightsail

1. Instancia Linux con Docker. Firewall de Lightsail: abrir **solo** 80 y 443; el 22 solo para las IP
   del equipo. **Nunca** abrir 6379 ni 6060.
2. Copiar el repositorio a `/opt/nasa-app-api` y crear `deploy/.env`.
3. `docker compose -f deploy/docker-compose.yml up -d --build`.
4. Pronóstico mensual:

   ```bash
   sudo cp deploy/systemd/nasa-forecast.* /etc/systemd/system/
   sudo systemctl enable --now nasa-forecast.timer
   ```

5. Diagnóstico (pprof y expvar) por túnel SSH: `ssh -L 6060:127.0.0.1:6060 usuario@servidor` y abrir
   `http://localhost:6060/debug/pprof/`.

## Comprobar que funciona

```bash
curl https://$API_DOMAIN/readyz
curl "https://$API_DOMAIN/v1/plan?lat=14.30&lon=-88.60&currentRotation=maize,beans"
curl "https://$API_DOMAIN/v1/departments/HN-LE/plan"
```
