# Fuentes de las constantes y supuestos del código

El código no lleva comentarios (ver `.claude/CLAUDE.md` § Comentarios). Este documento registra de
dónde sale cada constante científica o agronómica y qué supuestos hace el motor. Si se cambia un
valor en el código, hay que cambiarlo acá.

Confianza: **A** = publicación verificada; **B** = fuente experta o valor A sin verificar;
**C** = estimación provisional (ver plan § 5.5).

## Evapotranspiración (`internal/agro/et0.go`)

| Constante | Valor | Fuente | Confianza |
| --------- | ----- | ------ | --------- |
| `LatentHeatOfVaporization` | 2,45 MJ/kg | FAO-56, ec. 3 (valor redondeado) | A |
| `solarConstant` | 0,0820 MJ m⁻² min⁻¹ | FAO-56, ec. 21 | A |
| `stefanBoltzmann` | 4,903 × 10⁻⁹ MJ K⁻⁴ m⁻² día⁻¹ | FAO-56, ec. 39 | A |
| `referenceAlbedo` | 0,23 | FAO-56, ec. 38 | A |
| `megajoulesToMillimeter` | 0,408 | FAO-56, 1/λ | A |
| Penman-Monteith | ec. 6 | FAO-56; probado contra el ejemplo 18 (ET0 = 3,9 mm/día) | A |
| Radiación extraterrestre | ec. 21 a 25 | FAO-56; probado contra el ejemplo 8 (Ra = 32,2) | A |
| Presión atmosférica | ec. 7 | FAO-56; probado contra el ejemplo 2 (81,8 kPa a 1800 m) | A |
| Hargreaves | ec. 52 | FAO-56; probado contra el cálculo manual con los datos del ejemplo 18 | A |

## Suelo (`internal/agro/soil.go`, `profile.go`)

| Elemento | Fuente | Confianza |
| -------- | ------ | --------- |
| Ecuaciones de θ1500 y θ33 | Saxton y Rawls (2006), ec. 1 y 2. El test compara contra el valor calculado a mano, no contra la tabla del artículo | B |
| MO = SOC × 1,724 | Factor de Van Bemmelen | B |
| Zona radicular: 50 % capa 0-30 cm y 50 % capa 30-60 cm | Supuesto del proyecto | C |
| Clases de textura | Triángulo USDA simplificado | B |

## Balance hídrico (`internal/agro/waterbalance.go`)

| Supuesto | Valor | Fuente | Confianza |
| -------- | ----- | ------ | --------- |
| Raíz a profundidad máxima desde la siembra | Zr de la ficha | Simplificación de FAO-56 (que la hace crecer) | B |
| Agotamiento inicial | 50 % del agua disponible, ajustado con 30 días de suelo desnudo antes de la siembra (Kc 0,3) | Supuesto del proyecto | C |
| Ajuste de p | p + 0,04 (5 − ETc), limitado a 0,1–0,8 | FAO-56, nota de la Tabla 22 | A |
| Riego | Repone todo el agotamiento cuando supera el agua fácilmente disponible | Supuesto del proyecto | B |
| Mulch (Quesungual) | ×1,1 agua disponible; ×0,4 factor C | Supuesto del proyecto | C |
| Umbral de pérdida | Rendimiento relativo < 0,5 | Plan § 7.4, a validar | C |

## Rendimiento (`waterbalance.go`, `crop.go`)

| Elemento | Fuente | Confianza |
| -------- | ------ | --------- |
| 1 − Ya/Ym = Ky (1 − ETa/ETc) | FAO-33 | A |
| Fases FAO-33 sobre etapas FAO-56: vegetativa = inicial + desarrollo; floración = primer 40 % de la etapa media; formación = resto de la etapa media; maduración = etapa final | Supuesto del proyecto | C |
| Combinación multiplicativa de fases | Doorenbos y Kassam (FAO-33) | B |

## Temperatura y ciclo (`crop.go`, `series.go`)

| Elemento | Valor | Fuente | Confianza |
| -------- | ----- | ------ | --------- |
| Gradiente térmico | 6,5 °C por 1000 m | Atmósfera estándar | A |
| Temperatura de referencia de las etapas de la ficha | 24 °C | Supuesto: tablas FAO-56 de zonas tropicales | C |
| Escala del ciclo | (24 − Tbase) / (Tmedia − Tbase), limitada a 0,7–1,8 | Grados-día, supuesto del proyecto | C |

## Indicadores climáticos (`indicators.go`, `quality.go`)

| Elemento | Valor | Fuente | Confianza |
| -------- | ----- | ------ | --------- |
| Inicio de lluvias | ≥ 20 mm en 3 días después del 1 de abril, sin racha seca > 7 días en los 30 siguientes | Tipo Sivakumar (1988) | B |
| Día seco | Lluvia < 0,5 × ET0 (en lugar de < 1 mm, por el sesgo de llovizna de MERRA-2) | Supuesto del proyecto, plan § 7.5 | C |
| Canícula | Período del 15 de junio al 31 de agosto con lluvia de 10 días < 0,5 × ET0 de 10 días, ≥ 5 días | Plan § 7.5, a calibrar | C |
| Humedad para sembrar en secano | Mediana de lluvia de la quincena ≥ 0,5 × ET0 | Supuesto del proyecto | C |
| Control de calidad | Mes > Q3 + 3 × RIC del mismo mes calendario en la celda | Plan § 10.1 | B |

## Pronóstico (`forecast.go`, `weights.go`, `internal/etl/forecast.go`)

| Elemento | Fuente | Confianza |
| -------- | ------ | --------- |
| Anomalía de cada miembro = total del miembro − (media del ensamble − anomalía publicada) | Open-Meteo publica la anomalía de la media frente a la climatología del modelo | B |
| Límites del tercil = ±0,43 σ, con σ = desvío interanual de POWER para ese mes | Terciles de una distribución normal | B |
| Ponderación de años análogos por tercil | Plan § 6.1 | B |

## Suelo y erosión en el puntaje (`score.go`)

| Elemento | Fuente | Confianza |
| -------- | ------ | --------- |
| Erosividad mensual = P² / P anual (índice de Fournier modificado por mes) | Arnoldus (1980) | B |
| Mes lluvioso = mediana ≥ 100 mm | Supuesto del proyecto | C |
| Pesos agua/suelo por prioridad: balanced 0,55/0,45; water 0,75/0,25; soil 0,30/0,70. Sin subpuntaje de ingreso (alcance solo clima y suelo, 2026-11-15) | `docs/calibracion.md` § 1 | C |
| Normalización de nitrógeno: (N + 60) / 180 | Supuesto del proyecto | C |
| Penalización por riego: 1 punto cada 40 mm, máximo 25 | Supuesto del proyecto | C |

## Reglas de rotación (`engine.go`, `crop.go`)

| Regla | Valor | Fuente | Confianza |
| ----- | ----- | ------ | --------- |
| Fechas de siembra candidatas | Inicio de cada quincena de calendario (día 1 y día 16), desde el día en que el terreno queda libre | Coherente con la climatología por quincena del perfil | B |
| Relevo de cobertura (`relay.guest = cover`) | Solo dentro de cultivos con `relay.hosts` que incluye `cover` (maíz y sorgo), desde 30 días después de la siembra hasta la cosecha | Decisión de alcance del 2026-11-15 (`docs/calibracion.md` § 3) | C |
| Relevo de grano (`relay.guest = grain`, frijol) | Solo dentro del maíz, desde el inicio de su etapa final (madurez, "doblado") hasta la cosecha | Práctica común en Honduras: Guillén (2024), *Guía de buenas prácticas para el manejo del cultivo de frijol en Honduras*, AgriLAC Resiliente, p. 8; DICTA (2019), *Guía práctica para cultivar frijol*. ICTA (2022), *Manual para producción de frijol*, ubica la siembra en la madurez fisiológica del maíz, más tarde que el inicio de la etapa final que usa el motor: calibrar con NDVI en la etapa 2 | B (práctica), C (momento) |
| Simulación del cultivo en relevo | Se simula como cultivo solo, sin competencia por agua con el cultivo anfitrión | Simplificación del proyecto | C |
| Plantillas | `relay: true` obliga el relevo; `false` deja que el motor elija entre siembra secuencial y relevo | Supuesto del proyecto | — |
| Rotaciones generadas | Deben incluir al menos un cultivo de cosecha (sin ella ganaba la cobertura sola) | Decisión del usuario, 2026-11-15 | — |
| Ciclo del maíz | Etapas 18/31/36/27 días a 24 °C (112 días): FAO-56 Tabla 11 acortada para reproducir la primera (siembra mayo-junio, cosecha septiembre) a ~1.000 m | Ajuste provisional; calibrar con NDVI en la etapa 2 | C |

## Ficha de cultivos

La genera `tools/gencatalog/gen.js` y queda en `internal/catalog/crops_data.go`. Cada valor lleva su
`source` y su `confidence` en el JSON. Para cambiar la ficha: editar `gen.js` y correr
`node tools/gencatalog/gen.js > internal/catalog/crops_data.go`. El validador rechaza cualquier
valor sin fuente.
