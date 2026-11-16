# nasa-app-api

Go API that projects 12-month crop rotations for every municipality in Honduras, based on climate
and soil conditions. Built for the **Field Shift** challenge of NASA Space Apps 2026.

Recommendations use climate and soil only. Prices, costs, pests, markets and official yields are out
of scope.

## How it works

- An ETL process (`cmd/etl`) downloads the source data, builds a climate and soil profile for each
  of the 299 municipalities, and publishes it to Redis as a versioned dataset.
- The API (`cmd/api`) reads only from Redis. No user request calls an external service.
- The engine (`internal/agro`) simulates the FAO-56 water balance of each candidate rotation using
  historical climate and the seasonal forecast, then scores it. Responses carry structured reasons;
  the frontend turns them into text.
- The user's latitude and longitude are used only to find the municipality. They are neither stored
  nor logged.

## Endpoints

| Method and path | Returns |
| --------------- | ------- |
| `GET /v1/plan?lat=&lon=` | Plan for the municipality that contains the point |
| `GET /v1/municipalities/{code}/plan` | Plan for one municipality |
| `GET /v1/departments/{code}/plan` | Aggregated plan for one department |
| `GET /v1/municipalities`, `GET /v1/departments` | Listings |
| `GET /v1/crops` | Catalog of the 22 crops |
| `GET /livez`, `GET /readyz` | Health checks |
| `GET /docs`, `GET /openapi.yaml` | Swagger UI and the OpenAPI 3.1 contract |

Optional plan parameters: `currentRotation`, `currentCrop`, `fieldFreeFrom`, `irrigation` and
`priority`. See `api/openapi.yaml` for details.

The API is public, requires no login, and is rate limited per IP. Errors follow RFC 9457
(`application/problem+json`). Response texts are in Spanish.

## Run and test

Requires Go 1.27 and, to run the API, Redis 8 (or Docker).

```bash
go test -race ./...
golangci-lint run ./...
govulncheck ./...
```

Local Docker environment, data loading and deployment: `docs/operacion.md`.

## Documentation

The documents are written in Spanish.

| Document | Content |
| -------- | ------- |
| `docs/plan-implementacion.md` | Scope, scientific basis and decisions |
| `docs/arquitectura.md` | Packages, flows, Redis data, concurrency and resilience |
| `docs/fuentes.md` | Source of every scientific and agronomic constant |
| `docs/operacion.md` | Data loading, deployment and verification |
| `docs/calibracion.md` | Engine calibration and scope decisions |

## Data sources and attribution

| Source | Use | Terms |
| ------ | --- | ----- |
| [NASA POWER](https://power.larc.nasa.gov/) (NASA Langley Research Center) | Historical daily climate | Public data; cite NASA POWER |
| [SoilGrids](https://soilgrids.org/) (ISRIC) | Soil properties | CC BY 4.0 |
| [geoBoundaries](https://www.geoboundaries.org/) | Municipal and department boundaries | Municipalities CC BY 4.0 (ICF); departments ODbL (OpenStreetMap) |
| [Open-Meteo](https://open-meteo.com/) (ECMWF SEAS5, Copernicus DEM) | Seasonal forecast and elevation | CC BY 4.0; the free API is for non-commercial use only |
| GPM IMERG (NASA GES DISC) | Calibration only (`cmd/calibrate`) | Public data; cite if results are published |

The files in `testdata/` are derived from these sources and are subject to the same terms.

## License

This repository has no open source license. The code is published for reference only: all rights
are reserved by its authors, and copying, modifying or redistributing it requires permission.
Third-party data keeps its own licenses (see the table above).

The skills in `.claude/skills/golang-*` are third-party and carry their own license
(`.claude/skills/LICENSE-samber-cc-skills-golang`).
