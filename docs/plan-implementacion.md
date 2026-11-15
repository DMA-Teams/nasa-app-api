# Plan de implementación: API de rotación de cultivos (Field Shift, NASA Space Apps 2026)

> Estado: **borrador v2, para revisión**. Todavía no hay código.
>
> v1 (2026-11-14): investigación inicial, base científica y decisiones abiertas D1 a D8.
> v2 (2026-11-14): incorpora las respuestas a D1 a D8, la ficha de propiedades de cultivos, el
> pronóstico estacional a 6 meses, el almacenamiento en Redis y Supabase, la precarga de Honduras y
> el despliegue en Lightsail.
>
> v3 (2026-11-14): incorpora las respuestas a P1 a P8 (§ 0.1) y deja las dudas restantes en § 12.
> v4 (2026-11-14): proyección por **municipio o departamento** en lugar de parcela, pronóstico
> estacional con ECMWF SEAS5 y **actualización periódica** del plan (§ 0.2). Cuando una sección
> anterior contradice § 0.2, manda § 0.2.
> v5 (2026-11-14): stack **Go** en lugar de .NET, departamento agrupado por recomendación,
> altitud municipal aprobada y año en curso en fase 2 (§ 0.3). Donde el documento dice .NET o C#,
> manda § 0.3.
>
> Las mediciones citadas se tomaron el 2026-11-14 contra las APIs públicas.

---

## 0.0 Estado actual

> **v6 (2026-11-15): la API está implementada y probada con datos reales de Lempira.** El estado y
> los pendientes están en `docs/estado.md`; la calibración del motor, en `docs/calibracion.md`.
> **Alcance final:** solo clima y suelo; se elimina el subpuntaje de ingreso (§ 4.9 y § 7.9 de este
> plan quedan reemplazados por `docs/calibracion.md` § 1).

---

## 0.1 Respuestas a P1 a P8 (v3)

| # | Respuesta | Efecto en el plan |
| - | --------- | ----------------- |
| P1 | Los trimestres se cuentan **desde la fecha en que se pide el plan**. Guardar los cultivos actuales del productor es **otro alcance** | Horizonte de 12 meses desde la fecha del pedido: T1 = días 1 a ~91, y así. Sin persistencia de datos del productor. Ver duda Q1 en § 12 |
| P2 | Supabase **en consideración**; Redis podría cubrirlo | MVP sin Supabase. Ver Q2 en § 12 |
| P3 | **Riego sí**, y la mayor cantidad posible de cultivos | El riego es un dato de entrada. El catálogo incluye todos los cultivos de § 5.2 y se amplía a medida que haya fuentes |
| P4 | Instancia de **4 GB** | Suelo a **250 m**, Lightsail Medium (24 USD al mes) |
| P5 | No se entendió la pregunta | Reformulada como Q3 en § 12 |
| P6 | **Solo las 2 fuentes** (NASA POWER y SoilGrids) | No se usa ERA5 ni CHIRPS. Los meses anómalos se detectan con el propio histórico de POWER y se **excluyen** (§ 10.1). Afecta al pronóstico: ver Q3 |
| P7 | Plantillas: **de momento no** | Ver Q4 en § 12 |
| P8 | **No hay revisión externa** | Todo valor que no sea confianza A sale marcado como baja confianza. Más cultivos implican más recomendaciones de baja confianza |

---

## 0.2 Respuestas a Q1 a Q6 y nuevo requisito (v4)

| # | Respuesta | Efecto en el plan |
| - | --------- | ----------------- |
| Q1 | Aprobado: cada trimestre proyecta qué se siembra, qué está en campo y qué se cosecha; la situación actual viene en el pedido, sin guardarse | § 6.2 sin cambios |
| Q2 | Aprobado: API pública, sin login, con límite por IP | Sin Supabase en el MVP |
| Q3 | Buscar una alternativa eficiente para el pronóstico | **ECMWF SEAS5 vía la API estacional de Open-Meteo** (§ 6.1, § 9.3) |
| Q4 | Razones estructuradas; el texto lo arma el frontend | La API no genera texto |
| Q5 | Pendiente | **Bloquea el inicio del código** (Q9) |
| Q6 | El usuario solo da su **ubicación** (se resuelve a municipio) o **elige un departamento o municipio**. Ningún otro dato territorial | La unidad de análisis pasa a ser el **municipio** (§ 8). Sin polígonos, sin altitud ni pendiente del usuario |
| Nuevo | El plan se **actualiza** con el tiempo: si cambian las condiciones proyectadas, cambia la recomendación | Ciclo de actualización en § 6.3 |

---

## 0.3 Respuestas a Q6 a Q9 (v5)

| # | Respuesta | Efecto en el plan |
| - | --------- | ----------------- |
| Q6 | Plan por municipio, **agrupado por recomendación** | Para un departamento se calcula cada municipio y se agrupan los que tienen la misma recomendación (§ 0.3.1) |
| Q7 | Aprobado | ETL único de altitud municipal con Copernicus DEM vía Open-Meteo (§ 0.3.2) |
| Q8 | Aprobado: fase 2 | El MVP actualiza el plan por la fecha y por el pronóstico mensual |
| Q9 | **Go** | Stack y estructura en § 0.3.3 |
| Futuro | Supabase para guardar planes, cultivos del productor y territorio, más adelante | Fuera del MVP. Cuando entre, la comparación "tu plan cambió" puede pasar del frontend al backend |

### 0.3.1 Departamento agrupado por recomendación

1. Se calcula el plan de cada municipio del departamento.
2. Se agrupan los municipios cuya **secuencia recomendada de cultivos por trimestre es la misma**.
3. Cada grupo trae la recomendación, la lista de municipios y el **rango** de sus métricas (por
   ejemplo, probabilidad de pérdida entre 10 % y 18 %), porque la misma secuencia puede tener
   distinto riesgo en cada municipio.

Costo: Olancho tiene 23 municipios, así que son 23 evaluaciones por pedido. Muchos comparten celda
de clima, y el perfil de cada municipio ya viene precalculado (§ 8.5). Se mide en la fase 2.

### 0.3.2 Altitud municipal

**Por qué importa:** la temperatura baja ~6,5 °C por cada 1.000 m de altura. POWER da la
temperatura de la **altura promedio de su celda** (1.021 m en el ejemplo). Si el municipio está a
300 m, en realidad hace ~4,7 °C más de lo que dice POWER; si está a 1.800 m, ~5 °C menos. Eso
cambia qué se puede sembrar:

- La papa necesita frío: sin corrección, el sistema podría recomendarla en un valle caliente.
- El frijol pierde flores con noches calientes.
- El maíz madura según el calor acumulado (grados-día): a 1.800 m su ciclo se alarga y puede no
  alcanzar a cosecharse antes de que terminen las lluvias.

**En la práctica:**

- **Qué se ejecuta:** se reparten puntos cada ~5 km dentro de cada municipio (con un mínimo por
  municipio, para que los pequeños no queden con uno o dos). Por cada punto se pide su altura a la
  API de elevación de Open-Meteo (modelo Copernicus DEM, 90 m), en lotes de 100 puntos por llamada.
- **Cuándo y cuántas veces:** **una sola vez**, en el ETL. El relieve no cambia.
- **Volumen:** ~4.500 puntos, ~45 llamadas (cálculo, no medido). En Redis, por municipio: altura
  mediana, mínima y máxima. Pocos KB.
- **Con quién compite:** API pública de Open-Meteo, solo durante el ETL.
- **Qué pasa si crece:** no crece.

**Cómo se usa:** T_municipio = T_celda − 0,0065 × (altura_mediana_municipio − altura_celda). Si el
rango del municipio es grande (por ejemplo, de 300 a 2.000 m), la respuesta lo advierte: "el plan
corresponde a la altura mediana del municipio (1.150 m)".

### 0.3.3 Stack: Go

Go es una buena elección para este proyecto. Pero dos de los motivos que diste no son exactos, y
conviene corregirlos ahora para no esperar del lenguaje algo que no da.

**Lo que Go sí aporta acá:**

- Un **binario único** y una imagen de Docker muy chica.
- **Poca memoria**: decenas de MB contra los cientos de MB típicos de .NET. Eso pesa en una
  instancia de Lightsail.
- Arranque instantáneo, concurrencia simple (goroutines) y una librería estándar suficiente para
  HTTP, JSON, logging (`slog`) y tests.

**Lo que hay que matizar:**

- **"Eficiencia":** el cálculo pesado son ciclos numéricos (balance hídrico). En eso Go y .NET 10
  rinden parecido; ninguno es un cuello de botella para este volumen. La ventaja real de Go es la
  memoria, no la velocidad de cálculo.
- **"Que no se caiga":** el lenguaje no lo garantiza. En Go, un `panic` dentro de una goroutine
  lanzada por el código **tira el proceso entero**: `net/http` recupera los panics del handler,
  pero no los de goroutines propias. La disponibilidad sale de:
  - un middleware de recuperación;
  - `context` con timeout en toda llamada a Redis;
  - límite de pedidos por IP;
  - un health check;
  - política de reinicio automático del contenedor (`restart: unless-stopped`);
  - límite de memoria del contenedor.
- **"Parecido a C#":** en la sintaxis y el tipado estático sí. En el modelo, no:
  - errores como valores, no excepciones;
  - interfaces implícitas, sin clases ni herencia;
  - genéricos más limitados, sin LINQ;
  - sin contenedor de inyección de dependencias por convención: el armado se hace a mano en
    `main`;
  - goroutines en lugar de `async/await`.

  La Clean Architecture de C# (cuatro proyectos, DTOs por capa, un mediador) se siente forzada en
  Go. Se mantiene la idea (el dominio no conoce Redis ni HTTP), con menos ceremonia.

**Estructura propuesta:**

```
cmd/api/            arranque de la API (armado de dependencias, servidor HTTP)
cmd/etl/            precarga: POWER, suelo agregado, altitud, límites, pronóstico mensual
internal/agro/      dominio puro: ET0, balance hídrico, Ky, RUSLE, aptitud, rotaciones, puntaje
internal/catalog/   ficha de cultivos (JSON embebido) y su validador
internal/plan/      casos de uso: perfil municipal, evaluación, agrupación por departamento
internal/geo/       límites, punto en polígono, celdas de POWER
internal/store/     adaptador de Redis
internal/sources/   clientes de POWER, SoilGrids, Open-Meteo (estacional y elevación)
internal/httpapi/   handlers, validación, errores (RFC 9457), límite por IP
```

- Las interfaces se definen **en el paquete que las usa** (por ejemplo, `plan` define qué necesita
  leer y `store` lo implementa). Es el estilo idiomático de Go.
- Dependencias externas mínimas: cliente de Redis (`go-redis`) y una librería de geometría para
  punto en polígono (candidata: `paulmach/orb`). El resto, con la librería estándar (`net/http`
  ya enruta por método y ruta).
- Pruebas con `go test`: el dominio contra los ejemplos de FAO-56, y los clientes contra
  respuestas grabadas.
- Los pasos con rásteres (reproyección y recorte de SoilGrids) siguen con **GDAL en Docker**,
  fuera de Go. El ETL de Go lee los binarios resultantes.
- El ETL de suelo conviene correrlo en una máquina de desarrollo, no en Lightsail: procesa cientos
  de MB de rásteres y en Lightsail solo hace falta cargar los agregados a Redis.

**Memoria en Lightsail con Go:** Redis ~100 MB + API ~30 a 60 MB + sistema operativo y proxy. La
instancia de 2 GB sobra; la de 4 GB (P4) deja todavía más margen. La decisión queda en el equipo.

---

## 0. Decisiones tomadas

| # | Decisión | Fuente |
| - | -------- | ------ |
| D1 | El plan anual se presenta **por trimestre y por semestre** | Respuesta del equipo |
| D2 | **30 años** de histórico de clima, precargados en **Redis** | Respuesta del equipo |
| D3 | Suelo de SoilGrids **precargado en Redis** | Respuesta del equipo |
| D4 | **Redis** para todos los datos de cálculo; **Supabase** para lo demás (ver P2, § 12) | Respuesta del equipo |
| D5 | **Precarga de todo Honduras** para la demo | Respuesta del equipo |
| D6 | Catálogo ampliado a cultivos que se siembran en Honduras (§ 5) | Respuesta del equipo |
| D7 | Explicaciones generadas con **plantillas** (propuesta; ver § 9.4) | Pendiente de confirmar |
| D8 | **No hay agrónomo** en el equipo: plan de mitigación en § 5.6 | Respuesta del equipo |
| D9 | Despliegue en **AWS Lightsail**, con Redis en la misma instancia | Respuesta del equipo |
| D10 | Horizonte de 12 meses: **6 meses con pronóstico estacional + 6 con climatología** (§ 6) | Respuesta del equipo, con matices |

---

## 1. Qué pide el reto y qué estamos construyendo

### 1.1 Lo que pide el reto (Field Shift, reto 7 del starter)

- Herramienta de **apoyo a la decisión**, no un agrónomo automático. El brief lo dice de forma
  explícita: la salida es para discutirla con un técnico local.
- Comparar **al menos tres rotaciones, incluida la actual**, y explicar cada una: razonamiento,
  evidencia, beneficios esperados, información faltante y riesgos.
- Persona de referencia: productor de maíz y frijol del **Corredor Seco** (Lempira, Choluteca),
  afectado por una **canícula** cada vez más larga e impredecible.
- Opciones sugeridas: maíz-frijol tradicional, maíz con sorgo (maicillo), maíz con cultivo de
  cobertura (mucuna o frijol) y el sistema agroforestal **Quesungual**.
- Salida: plan de siembra por temporada (primera y postrera). Nosotros lo presentamos además por
  trimestre y semestre (D1).
- Nivel "Developer": índice de déficit hídrico (lluvia menos evapotranspiración) **por quincena**,
  cruzado con el calendario de cada rotación, y explicaciones generadas a partir de los cálculos.
- Error frecuente señalado por el brief: usar datos de EE. UU. (SSURGO) en Honduras. Para suelo hay
  que usar SoilGrids o FAO GAEZ y **declarar la fuente y su cobertura**.

### 1.2 Lo que proponemos construir

Una API que, dada una ubicación (punto o polígono), la rotación actual del productor y su prioridad:

1. Construye un **perfil del sitio**: clima histórico de 30 años resumido por quincena, suelo por
   capas y calidad de los datos.
2. Cruza ese perfil con la **ficha de propiedades de cada cultivo** (§ 5) para saber qué cultivos
   son viables, cuándo, y con qué riesgo.
3. Evalúa **rotaciones candidatas** simulando, año por año del histórico, el balance hídrico de
   cada cultivo en su ventana de siembra. Los primeros 6 meses ponderan los años según el
   pronóstico estacional vigente (§ 6).
4. Devuelve, por trimestre y semestre, qué se siembra, qué está en campo, qué se cosecha y cómo
   queda el suelo, con **probabilidades** e indicadores de salud del suelo.
5. Explica cada recomendación con el número que la respalda y la fuente de ese número.

---

## 2. Correcciones a supuestos iniciales (verificadas con datos)

### 2.1 Estacionalidad de la lluvia

Climatología de POWER para lat 14.30, lon -88.60 (1995–2024, calculada desde la serie diaria):

| Mes | Lluvia P50 (mm) | Lluvia P20 (mm) | ET0 Hargreaves (mm) |
| --- | --------------- | --------------- | ------------------- |
| Ene | 25  | 15  | 107 |
| Feb | 17  | 8   | 121 |
| Mar | 15  | 8   | 159 |
| Abr | 30  | 17  | 165 |
| May | 126 | 77  | 152 |
| Jun | 197 | 153 | 125 |
| Jul | 138 | 99  | 132 |
| Ago | 173 | 133 | 130 |
| Sep | 206 | 150 | 114 |
| Oct | 154 | 93  | 107 |
| Nov | 54  | 26  | 95  |
| Dic | 34  | 19  | 97  |

- Enero–marzo es la **estación seca** en el occidente y el sur. Sin riego, ahí no se siembra.
- Lluvias de **mayo a octubre**, con la caída de **julio** (canícula).
- En el **Litoral Atlántico** sí llueve en invierno y existe el ciclo de **apante** (frijol de
  noviembre a enero).
- **Con riego** (Choluteca, Valle, Comayagua) el primer trimestre sí se usa: melón, sandía,
  hortalizas y arroz.

**Consecuencia de diseño:** el calendario se deriva de los datos de cada sitio, no de una regla
nacional. El mismo motor produce "primera + postrera" en Lempira y "primera + postrera + apante"
en Atlántida.

### 2.2 Gramíneas y leguminosas

El arroz, el maíz y el sorgo son **gramíneas (Poaceae)**: extraen nitrógeno y no lo fijan. Las
**leguminosas** (frijol, maní, caupí, mucuna, gandul, canavalia) lo reponen por fijación biológica.
Rotar gramínea con gramínea no corta ciclos de plagas ni repone nitrógeno. Esto se modela con la
familia botánica y el balance de nitrógeno de cada cultivo (§ 5).

### 2.3 Resolución espacial: las celdas son más grandes que la finca

| Fuente | Resolución nativa | Tamaño en Honduras (lat ~14°) | Verificado |
| ------ | ----------------- | ----------------------------- | ---------- |
| NASA POWER, meteorología (MERRA-2) | 0,5° lat × 0,625° lon | ~55 km × ~67 km (~3.700 km²) | Sí, con llamadas reales |
| NASA POWER, radiación solar | 1° × 1° | ~111 km × ~108 km | Según documentación |
| SoilGrids 2.0 | 250 m | 6,25 ha por píxel | Sí, en el VRT publicado |

Prueba del 2026-11-14 contra POWER: lat 14.30 y lat 14.45 (misma longitud, 16 km de distancia)
devuelven **valores idénticos** y la misma elevación de celda (1021,41 m). Con lon -88.40 los
valores cambian (celda vecina, 1110 m).

Una finca típica cae dentro de un solo píxel de suelo y de una sola celda de clima. El riesgo es el
inverso: la celda promedia valles y montañas. Ver § 8.4 (corrección por altitud).

### 2.4 Pronóstico a 6 meses: qué se puede y qué no

Se puede "proyectar" hasta unos 6 meses, pero **no el tiempo diario**: lo que existe a ese plazo son
**pronósticos estacionales probabilísticos**, que dicen cosas como "para mayo a julio hay 50 % de
probabilidad de lluvia bajo lo normal, 35 % normal y 15 % sobre lo normal". El tiempo día a día
solo tiene buena exactitud hasta 7 a 10 días.

Además, la exactitud a 6 meses **depende de la temporada** en Centroamérica:

- Kowal y colaboradores (comparación de NMME y Copernicus C3S contra CHIRPS, 1994–2016) reportan
  **menor habilidad en mayo-junio-julio** (inicio de lluvias, o sea la **primera**) y **alta
  habilidad en agosto-septiembre-octubre** en el Corredor Seco (la **postrera**).
- Una evaluación de NOAA CPC de los modelos NMME encontró **sesgo seco** sobre Guatemala, El
  Salvador, Honduras y el norte de Nicaragua.

Por eso el plan **no reemplaza** la climatología con el pronóstico: la **pondera** (§ 6).

Dato relevante para el plan 2027: el índice ONI de NOAA marca **El Niño fuerte en curso**
(ASO 2026 = +2,46 °C, consultado el 2026-11-14). En el histórico de POWER, la racha seca más larga
de junio a agosto en el punto de ejemplo fue en **2015** (12 días), otro año de El Niño fuerte.

---

## 3. Fuentes de datos

### 3.1 NASA POWER (API diaria por punto, comunidad AG)

| Parámetro | Unidad | Uso |
| --------- | ------ | --- |
| `PRECTOTCORR` | mm/día | Lluvia: balance hídrico, inicio de lluvias, canícula |
| `T2M`, `T2M_MAX`, `T2M_MIN` | °C | ET0, grados-día, estrés térmico, vernalización y altitud |
| `RH2M`, `T2MDEW` | %, °C | Presión de vapor real (Penman-Monteith); ajuste climático de Kc |
| `WS2M` | m/s | Viento a 2 m (Penman-Monteith); ajuste climático de Kc |
| `ALLSKY_SFC_SW_DWN` | MJ/m²/día | Radiación solar (Rs) |
| `TOA_SW_DWN` | MJ/m²/día | Radiación extraterrestre (Ra), para Hargreaves y Rso |
| `PS` | kPa | Constante psicrométrica |
| `GWETROOT` | 0–1 | Humedad del suelo del modelo en zona radicular. Solo validación cruzada |
| `EVPTRNS` | MJ/m²/día | Evapotranspiración real del modelo. Solo referencia |

- `EVPTRNS` es **energía**, no milímetros (se divide por 2,45 MJ/kg para pasar a mm/día) y es la
  evapotranspiración **real** del modelo, no la ET0 de FAO. La demanda de los cultivos se calcula
  con ET0 (§ 7.1).
- En `temporal/climatology`, `T2M_MAX` mensual es el **máximo extremo** del período, no el promedio
  de las máximas diarias. Todo se calcula desde la serie diaria.
- Rango propuesto: **1996-01-01 a 2025-12-31** (30 años completos; 2025 ya cerró).

### 3.2 SoilGrids 2.0 (ISRIC)

Propiedades: `clay`, `sand`, `silt`, `soc`, `nitrogen`, `phh2o`, `cec`, `bdod`. Profundidades:
`0-5cm`, `5-15cm`, `15-30cm`, `30-60cm`. Valores: `mean`, `Q0.05`, `Q0.95`. Los valores vienen
como enteros escalados y se dividen por `d_factor` (ejemplo: `clay` 359 → 35,9 %).

Problemas medidos:

- La **API REST** tardó 6,9 s y 22,4 s; el límite de uso justo es **5 llamadas por minuto**; está
  en beta.
- **No sirve para precargar Honduras**: Honduras tiene ~1,8 millones de píxeles de 250 m. A 5
  llamadas por minuto serían **360.000 minutos, unos 250 días**.
- La precarga se hace desde los **rásteres publicados por WebDAV**
  (`files.isric.org/soilgrids/latest/data/`), que están en **proyección Interrupted Goode
  Homolosine** (verificado en `clay_0-5cm_mean.vrt`), no en latitud/longitud. Hay que reproyectar.
- **Incertidumbre grande**: la arcilla de 0–5 cm del punto de ejemplo tiene media 35,9 % y su
  intervalo de predicción al 90 % va de **4,2 % a 72,1 %**.

### 3.3 Índices y pronósticos estacionales (§ 6)

| Fuente | Qué da | Formato | Uso |
| ------ | ------ | ------- | --- |
| NOAA CPC, índice ONI | Anomalía de temperatura del Niño 3.4 por trimestre móvil, desde 1950 | Texto plano (`oni.ascii.txt`), verificado | Clasificar los 30 años históricos en El Niño, Neutro o La Niña |
| IRI / NOAA CPC, pronóstico ENSO | Probabilidad de El Niño, Neutro y La Niña para los próximos trimestres | Página web y datos del IRI; formato exacto **por confirmar** | Ponderar los años históricos (MVP) |
| NMME, incluido el modelo **NASA GEOS-S2S** | Probabilidades por tercil de lluvia y temperatura, hasta varios meses de anticipación | NetCDF vía IRI Data Library; **por confirmar** | Ponderación por terciles (fase 2) |
| Foro del Clima de América Central (CRRH-SICA) | Perspectiva regional por terciles para un trimestre, 3 veces al año | PDF con mapas | Contraste manual, no automatizable |

### 3.4 Fuentes complementarias (fase 2)

CHIRPS (lluvia a ~5,5 km, para validar POWER y calibrar la canícula), GPM IMERG, DEM (SRTM o
Copernicus GLO-30, para altitud y pendiente), SMAP y NDVI (validación), FAO GAEZ (contraste de
aptitud).

---

## 4. Cómo encajan las piezas

```
                    ┌──────────────────────── precarga única (ETL) ────────────────────────┐
NASA POWER ─────────┤ 88 celdas × 30 años                                                  │
SoilGrids WebDAV ───┤ 96 rásteres recortados a Honduras                                    ├──► Redis
NOAA ONI ───────────┤ clasificación ENSO de cada año                                       │
                    └──────────────────────────────────────────────────────────────────────┘
                    ┌──────────────── actualización mensual ─────────────┐
IRI / CPC ENSO ─────┤ probabilidades para los próximos 6 meses           ├──────────────► Redis
                    └────────────────────────────────────────────────────┘

Usuario ──► API ──► lee Redis (clima de la celda, suelo del píxel, pronóstico)
                ──► ficha de cultivos (§ 5)
                ──► motores de cálculo (§ 7)
                ──► guarda el resultado en Redis
                ──► guarda parcela y plan en Supabase (por confirmar, P2)
```

**Ninguna consulta de usuario llama a NASA ni a ISRIC.** Las APIs externas solo se tocan en la
precarga y en la actualización mensual del pronóstico.

---

## 5. Propiedades de los cultivos: la ficha técnica y cómo alimenta las fórmulas

Esta es la mitad del sistema que no viene de ninguna API: **lo que sabemos de cada cultivo**. Las
fórmulas de § 7 son genéricas; lo que las vuelve "maíz" o "frijol" son estos parámetros.

### 5.1 Qué propiedad entra en qué cálculo

| Propiedad del cultivo | Fórmula o regla donde entra | Qué decide |
| --------------------- | --------------------------- | ---------- |
| Duración del ciclo y de cada etapa (inicial, desarrollo, media, final) | Curva de Kc y ubicación en el calendario | Cuándo se siembra, cuándo se cosecha, qué trimestres ocupa |
| Kc ini, Kc med, Kc fin | ETc = Kc × ET0 (FAO-56) | Cuánta agua pide en cada etapa |
| Altura máxima | Ajuste climático de Kc med y Kc fin (FAO-56, ec. 62, con viento y humedad de POWER) | Corrige Kc para clima seco o ventoso |
| Profundidad de raíz (Zr) | Agua disponible ADT = (θCC − θPMP) × Zr | Cuánta agua del suelo puede usar |
| Fracción de agotamiento (p) | Agua fácilmente disponible AFA = p × ADT, y coeficiente de estrés Ks | Cuándo empieza a sufrir por sequía |
| Sensibilidad al déficit (Ky), total y **por etapa** | 1 − Ya/Ym = Ky × (1 − ETa/ETc) (FAO-33) | Cuánto rendimiento pierde por cada falta de agua; la floración suele ser la etapa más sensible |
| Temperaturas cardinales (base, óptima, máxima) | Grados-día y aptitud térmica | Duración real del ciclo según la altitud; si el lugar es demasiado frío o caliente |
| Umbral de estrés por calor en floración | Conteo de días con Tmax (o Tmin) sobre el umbral | Riesgo de aborto floral |
| Rango de altitud típico | Filtro de aptitud | Ejemplo: papa solo en altura |
| pH óptimo y absoluto | Aptitud (función trapezoidal) | Descarta suelos muy ácidos o alcalinos |
| Textura preferida y tolerancia al anegamiento | Aptitud con arcilla y arena de SoilGrids | El frijol y el ajonjolí no toleran encharcamiento; el arroz lo necesita |
| Requiere riego o inundación | Filtro de aptitud | El arroz y las hortalizas de verano solo con riego |
| Familia botánica | Regla de rotación: no repetir familia seguida | Corta ciclos de plagas y enfermedades |
| Fijación de nitrógeno (kg N/ha) | Balance de N de la rotación | Premia las leguminosas |
| Extracción de N por tonelada cosechada | Balance de N de la rotación | Penaliza los cultivos exigentes |
| Biomasa de residuos y relación C:N | Aporte de carbono al suelo | Tendencia de la materia orgánica |
| Factor C de RUSLE por etapa | Erosión mes a mes | Cuánto protege el suelo cuando llueve fuerte |
| Rendimiento de referencia en Honduras | Subpuntaje de ingreso | Base para comparar ingreso (débil, ver § 10.6) |
| Tipo (anual, semiperenne, perenne) | Generador de rotaciones | La yuca ocupa 3 a 4 trimestres; los perennes no rotan |

### 5.2 Valores de referencia por cultivo

Los valores de Kc, Zr y p fueron **verificados** contra FAO-56 (Tabla 12 y Tabla 22, consultadas el
2026-11-14). Los de Ky vienen de FAO-33 y **todavía no se verificaron** contra la publicación.

Sobre Kc ini: FAO-56 no lo trata como valor fijo para la mayoría de los cultivos de campo, porque
depende de cada cuánto se moja la superficie. Se calcula con el procedimiento de FAO-56 (capítulo 6)
a partir de la ET0 y la frecuencia de lluvia del sitio. Donde la tabla lo da, se usa como control.

| Cultivo | Familia | Kc ini | Kc med | Kc fin | Zr (m) | p | Ky total (FAO-33, verificar) | Fija N |
| ------- | ------- | ------ | ------ | ------ | ------ | - | ---------------------------- | ------ |
| Maíz grano | Poaceae | calcular | 1,20 | 0,35–0,60 | 1,0–1,7 | 0,55 | 1,25 | No |
| Sorgo grano (maicillo) | Poaceae | calcular | 1,00–1,10 | 0,55 | 1,0–2,0 | 0,55 | 0,90 | No |
| Arroz | Poaceae | 1,05 | 1,20 | 0,60–0,90 | 0,5–1,0 | 0,20 (de saturación) | sin dato | No |
| Frijol seco | Fabaceae | 0,40 | 1,15 | 0,35 | 0,6–0,9 | 0,45 | 1,15 | Sí |
| Maní | Fabaceae | calcular | 1,15 | 0,60 | 0,5–1,0 | 0,50 | 0,70 | Sí |
| Caupí (frijol de vara) | Fabaceae | calcular | 1,05 | 0,35–0,60 | 0,6–1,0 | 0,45 | sin dato | Sí |
| Soya | Fabaceae | calcular | 1,15 | 0,50 | 0,6–1,3 | 0,50 | 0,85 | Sí |
| Ajonjolí | Pedaliaceae | calcular | 1,10 | 0,25 | 1,0–1,5 | 0,60 | sin dato | No |
| Yuca (año 1) | Euphorbiaceae | 0,30 | 0,80 | 0,30 | 0,5–0,8 | 0,35 | sin dato | No |
| Camote | Convolvulaceae | calcular | 1,15 | 0,65 | 1,0–1,5 | 0,65 | sin dato | No |
| Papa | Solanaceae | calcular | 1,15 | 0,75 | 0,4–0,6 | 0,35 | 1,10 | No |
| Tomate | Solanaceae | calcular | 1,15 | 0,70–0,90 | 0,7–1,5 | 0,40 | 1,05 | No |
| Chile dulce | Solanaceae | calcular | 1,05 | 0,90 | 0,5–1,0 | 0,30 | 1,10 | No |
| Cebolla | Amaryllidaceae | calcular | 1,05 | 0,75 | 0,3–0,6 | 0,30 | 1,10 | No |
| Repollo | Brassicaceae | calcular | 1,05 | 0,95 | 0,5–0,8 | 0,45 | 0,95 | No |
| Pepino | Cucurbitaceae | 0,60 | 1,00 | 0,75 | 0,7–1,2 | 0,50 | sin dato | No |
| Sandía | Cucurbitaceae | 0,40 | 1,00 | 0,75 | 0,8–1,5 | 0,40 | 1,10 | No |
| Melón | Cucurbitaceae | 0,50 | 0,85 | 0,60 | 0,8–1,5 | 0,40 | sin dato | No |
| Ayote / calabaza | Cucurbitaceae | calcular | 1,00 | 0,80 | 1,0–1,5 | 0,35 | sin dato | No |

Ky por etapa (FAO-33, **verificar**), que es lo que permite decir "la canícula cayó en floración":

| Cultivo | Vegetativa | Floración | Formación del grano | Maduración |
| ------- | ---------- | --------- | ------------------- | ---------- |
| Maíz | 0,40 | 1,50 | 0,50 | 0,20 |
| Frijol | 0,20 | 1,10 | 0,75 | 0,20 |
| Sorgo | 0,20 | 0,55 | 0,45 | 0,20 |
| Maní | 0,20 | 0,80 | 0,60 | 0,20 |

**Sin cobertura en FAO-56 ni FAO-33** (confianza baja; requieren otra fuente): mucuna, canavalia,
gandul (frijol de palo), okra. En el MVP, mucuna y canavalia entran como **coberturas**, no como
cultivo de cosecha: importan su cobertura del suelo, su biomasa y su nitrógeno, no su rendimiento.

**Fuera del generador de rotaciones** (perennes): café, caña de azúcar, palma africana, banano y
plátano, cacao. No rotan por temporada. El sistema **Quesungual** sí entra, como sistema (cultivos
anuales bajo árboles dispersos con mulch permanente), no como cultivo.

### 5.3 Propiedades que faltan completar (con fuente)

Para cada cultivo todavía hay que llenar, con una fuente citada: temperaturas cardinales, umbral de
calor en floración, rango de altitud, pH, tolerancia al anegamiento, ciclos de las variedades que se
usan en Honduras, extracción y fijación de N, biomasa de residuos, factor C por etapa y rendimiento
de referencia.

Fuentes candidatas, en orden de preferencia:

1. **Locales**: DICTA (fichas de variedades: maíz DICTA Guayape, frijol Amadeus 77, sorgo DICTA
   10 BMR, entre otras), Zamorano (Programa de Investigaciones en Frijol), SAG.
2. **FAO**: FAO-56, FAO-33, AquaCrop (parámetros calibrados de maíz, sorgo y otros), Ecocrop
   (opinión experta), tablas de AEZ.
3. **Literatura revisada por pares** para coberturas y umbrales de calor.

### 5.4 Reglas de rotación (cualitativas, de principios agronómicos generales)

- No repetir **familia** en temporadas consecutivas; especialmente Solanaceae (tomate, chile, papa)
  y Cucurbitaceae, que comparten enfermedades de suelo.
- Después de una gramínea, preferir una **leguminosa**.
- Alternar raíces profundas (sorgo, maíz) con superficiales (cebolla, papa).
- No dejar el **suelo desnudo** durante los meses de lluvia más erosiva: cobertura o relevo.
- Arroz solo con riego o vega declarados; hortalizas de verano solo con riego.
- Las reglas son **configurables** en la ficha, con su fuente, igual que los números.

### 5.5 Nivel de confianza de cada valor

Cada valor de la ficha lleva una de tres marcas:

- **A**: tabla FAO o publicación revisada, verificada contra la fuente.
- **B**: Ecocrop, AquaCrop u otra fuente experta, o un valor A todavía sin verificar.
- **C**: estimación provisional, pendiente de fuente.

La API devuelve, en cada recomendación, **qué valores C intervinieron**. Una recomendación que
depende de valores C sale marcada como "baja confianza".

### 5.6 Sin agrónomo en el equipo (D8): mitigación

1. **Ningún valor sin fuente.** El validador de la ficha rechaza cualquier número sin `source` y
   sin marca de confianza.
2. **Pruebas de aceptación agronómica**: casos con resultado conocido que el sistema **debe**
   reproducir; si no, el motor está mal. Ejemplos:
   - No recomendar cultivos de secano en enero–marzo en el Corredor Seco.
   - Recomendar el frijol de postrera (agosto–septiembre) en el occidente.
   - Mostrar riesgo de canícula en el maíz de primera en Lempira y Choluteca.
   - Encontrar la ventana de apante en el Litoral Atlántico.
   - Penalizar la rotación maíz → sorgo → maíz frente a maíz → frijol en el puntaje de suelo.
3. **Revisión externa**: pedir una revisión corta de la ficha a DICTA, Zamorano, UNAG o FAO
   Honduras (ver P8, § 12).
4. Disclaimer en cada respuesta y lenguaje de probabilidades.

---

## 6. Horizonte de 12 meses: pronóstico y climatología

### 6.1 Cómo se usa el pronóstico: climatología ponderada

El motor **siempre** simula los 30 años del histórico. Lo que cambia el pronóstico es **cuánto pesa
cada año** al calcular las probabilidades:

- **Meses 1 a 6 (semestre 1):** cada año histórico recibe un peso según su parecido con el
  pronóstico.
  - **v4, con ECMWF SEAS5** (vía la API estacional de Open-Meteo): para cada celda y cada mes del
    horizonte, los 51 miembros del ensamble dan la **probabilidad** de que el mes salga seco,
    normal o húmedo **respecto de la climatología del propio modelo** (anomalía = pronóstico −
    climatología del modelo, que la API provee). Con eso, los años históricos de POWER del tercil
    seco, normal y húmedo de ese mes se reparten el peso según esas probabilidades.
  - Por qué con anomalías y no con valores crudos: SEAS5 **no está corregido por sesgo**, y los
    modelos tienen sesgo seco sobre Honduras. Comparar su lluvia cruda contra POWER mezclaría
    errores de dos modelos.
  - El método exacto para pasar de anomalías a probabilidades por tercil se cierra en la fase 1,
    con las respuestas reales de la API.
  - Descartado como fuente principal: las probabilidades oficiales de ENSO de CPC/IRI **solo se
    publican como tablas HTML** (no hay JSON ni CSV documentado). Leerlas implica parsear HTML,
    que se rompe cuando cambian la página. El índice ONI (texto plano) se sigue usando para
    etiquetar los años históricos en las explicaciones.
- **Meses 7 a 12 (semestre 2):** todos los años pesan igual (climatología pura). A ese plazo el
  pronóstico ya no aporta información confiable.

Ventajas:

- Un solo motor; el pronóstico no requiere fórmulas nuevas.
- Es explicable: "con El Niño, en los años parecidos del histórico el maíz de primera perdió más
  de la mitad en 4 de 7 años".
- No inventa precisión: si el pronóstico es incierto (33/33/33), el resultado vuelve a la
  climatología.

Límite: en 30 años hay pocos años de cada fase de ENSO (del orden de 7 a 10 por fase). Por eso se
**pondera** en lugar de **filtrar**: los demás años siguen contando, con menos peso.

### 6.2 Cómo se presentan el trimestre y el semestre (D1)

El motor trabaja por **quincena** porque los ciclos no calzan con los trimestres (el maíz de primera
se siembra en mayo y se cosecha en septiembre: cruza dos trimestres). La respuesta se **agrega** así:

- **Por trimestre** (T1 a T4): qué se siembra, qué está en campo, qué se cosecha, si el suelo queda
  cubierto o desnudo, el déficit hídrico esperado del trimestre y su riesgo.
- **Por semestre** (S1 y S2): el resumen de los dos trimestres, el balance de nitrógeno, la erosión
  relativa y el nivel de confianza del pronóstico (S1 ponderado por pronóstico, S2 climatológico).
- **Por temporada agrícola** (primera, postrera, apante): como vista adicional, porque es como se
  habla en el campo y lo que pide el reto.

El horizonte arranca en la **fecha del pedido** (P1): T1 son los primeros ~91 días desde esa
fecha, y así sucesivamente. Cada vez que se pide el plan, la ventana se mueve.

### 6.3 Actualización periódica del plan (v4)

Requisito: si en enero el plan sugiere algo para diciembre y en junio las condiciones proyectadas
cambian, la recomendación para diciembre tiene que cambiar.

**Cómo se cumple:** la API **no guarda planes**. Cada pedido recalcula el plan con lo último
disponible, así que el plan "se actualiza" cada vez que el productor lo abre. Lo que cambia entre un
pedido y otro:

1. **La fecha**: la ventana de 12 meses se corre.
2. **El pronóstico**: SEAS5 sale **una vez al mes** (el día 5). Cubre hasta 7 meses.
3. **Lo observado en el año en curso** (fase 2, ver Q8): cómo vienen las lluvias de este año,
   por ejemplo si el inicio de lluvias se atrasó.

**Qué meses puede mover el pronóstico.** Solo los que caen dentro de los ~7 meses que cubre SEAS5.
Con tu ejemplo:

| Fecha del pedido | ¿Diciembre está dentro del pronóstico? | Diciembre se calcula con |
| ---------------- | -------------------------------------- | ------------------------ |
| Enero | No (está a 11 meses) | Climatología de 30 años |
| Junio, con la corrida del 5 de junio | Sí (junio a diciembre son 7 meses) | Climatología ponderada por el pronóstico |
| Septiembre | Sí, y con un pronóstico más cercano | Climatología ponderada por un pronóstico más confiable |

Entonces la recomendación de diciembre **puede** cambiar desde junio, y se vuelve más confiable a
medida que diciembre se acerca. Si el pronóstico no se aparta de lo normal, la recomendación no
cambia, y eso también es información.

**Comparar con el plan anterior.** Como la API no guarda nada (Q1, Q2), mostrar "tu plan cambió"
le corresponde al **frontend**: guarda localmente el último plan y lo compara. Para eso la API
devuelve en cada plan la versión del pronóstico, la de los datos y la de la ficha de cultivos, y la
fecha de cálculo.

**Riesgo: recomendaciones que cambian por ruido.** Si dos rotaciones salen casi empatadas, un
pronóstico apenas distinto puede invertir el orden de un mes a otro, y el productor ve un plan que
"cambia de opinión". Mitigación: la API devuelve el **margen** entre la primera y la segunda opción
y el nivel de confianza; con un margen chico, el frontend lo presenta como empate.

---

## 7. Base científica: fórmulas

Criterio: **métodos estándar, publicados y explicables**. Cada número que aparezca en una
explicación tiene que poder rastrearse hasta una fórmula, un parámetro de cultivo y una fuente.

### 7.1 Demanda de agua (FAO-56)

- ET0 diaria con **FAO Penman-Monteith** (FAO-56, ec. 6). POWER entrega todas las entradas.
- **Hargreaves-Samani** (FAO-56, ec. 52) como control cruzado.
- ETc = Kc × ET0, con la curva de Kc por etapas y el ajuste climático de Kc med y Kc fin
  (FAO-56, ec. 62), usando la ficha de § 5.

### 7.2 Agua del suelo (pedotransferencia)

- **Saxton y Rawls (2006)**: θCC y θPMP desde arena, arcilla y materia orgánica (MO ≈ SOC × 1,724).
- ADT = 1000 × (θCC − θPMP) × Zr, ponderada por capa.
- Densidad aparente para detectar compactación y calcular el stock de carbono.
- La ADT también se calcula con los cuantiles Q0.05 y Q0.95 para ver si la recomendación cambia
  dentro de la incertidumbre de SoilGrids.

### 7.3 Balance hídrico diario (FAO-56, capítulo 8)

1. D_i = D_(i−1) − P_ef,i + ETc_aj,i (+ riego si la parcela lo tiene) (+ percolación sobre CC).
2. Ks = 1 si D ≤ AFA; si no, Ks = (ADT − D) / (ADT − AFA).
3. ETa = Ks × Kc × ET0.

Con riego (P3, § 12): el modelo agrega riego cuando D llega a AFA y reporta **cuántos milímetros de
riego** hicieron falta, que es un costo comparable entre rotaciones.

### 7.4 Riesgo de pérdida (FAO-33)

1 − Ya/Ym = Ky × (1 − ETa/ETc), **por etapa** con Ky por etapa cuando exista. Sobre los 30 años
(ponderados en S1) sale la distribución: rendimiento relativo esperado y **probabilidad de pérdida**
(años con Ya/Ym < 0,5; umbral a validar).

### 7.5 Inicio de lluvias y canícula

- **Inicio**: primer día después de una fecha mínima con ≥ 20 mm en 3 días y sin racha seca de
  más de 7 días en los 30 días siguientes (tipo Sivakumar). Da la ventana de siembra con su
  probabilidad.
- **Canícula por déficit**, no por días secos: por el sesgo de llovizna, POWER da rachas de solo
  2 a 12 días con menos de 1 mm entre junio y agosto, cuando la canícula real dura de 3 a 6 semanas.
  Se detecta como período continuo con lluvia de 10 días < 0,5 × ET0 de 10 días. El umbral se
  calibra contra CHIRPS o contra fechas conocidas.

### 7.6 Térmico

- **Grados-día** con la temperatura base de cada cultivo para ajustar la duración del ciclo según la
  altitud.
- **Estrés por calor en floración** con el umbral de cada cultivo.

### 7.7 Aptitud (tipo Ecocrop)

Funciones trapezoidales para temperatura, pH, textura, anegamiento y altitud; puntaje = **mínimo**
de los factores (ley del mínimo). Ecocrop solo filtra lo imposible; el agua se evalúa con § 7.3.

### 7.8 Salud del suelo

- **RUSLE** relativo: A = R × K × LS × C × P. R con el índice de Fournier modificado (lluvia
  mensual), K con la ecuación de Williams (EPIC) desde SoilGrids, LS desde la clase de pendiente
  del usuario (fase 2: DEM), **C mes a mes** según la rotación.
- Meses de suelo desnudo en temporada de lluvia.
- Balance de N: fijación menos extracción.
- Aporte de carbono por residuos.
- Diversidad de familias.

### 7.9 Puntaje

Suma ponderada transparente de tres subpuntajes (agua/riesgo, suelo, ingreso), con pesos según la
prioridad del productor. La respuesta muestra siempre los tres por separado.

### 7.10 Generación de rotaciones

1. **Filtro de aptitud**: con el perfil del sitio, se descartan los cultivos imposibles (altitud,
   temperatura, pH, riego).
2. **Ventanas viables**: para cada cultivo que pasa el filtro, las quincenas de siembra con agua
   suficiente.
3. **Encadenamiento**: secuencias de 12 meses que respetan las reglas de § 5.4. Con ~20 cultivos el
   espacio puede llegar a miles de secuencias; el filtro previo lo reduce a decenas o cientos.
4. **Simulación** de cada secuencia sobre 30 años y puntaje.
5. Respuesta: la rotación actual, las plantillas del reto (maíz-frijol, maíz-sorgo, maíz-mucuna,
   Quesungual) y las **mejores** encontradas.

Costo estimado: 1.000 secuencias × 30 años × 365 días ≈ 11 millones de pasos de balance, del orden
de 1 segundo en C#. **No medido**; se mide en la fase 2 del plan (§ 11).

---

## 8. Manejo espacial (v4: por municipio)

### 8.1 Entrada

El usuario da **una** de estas dos cosas:

- Su **ubicación** (latitud y longitud del teléfono). La API la usa solo para encontrar el
  municipio y **no la guarda ni la registra en logs**.
- Un **municipio** o un **departamento** elegido de una lista.

Más los datos no territoriales: cultivo actual y rotación habitual, riego sí o no, y prioridad.

### 8.2 Límites administrativos

Archivo estático versionado en el repositorio, sin API. Candidato verificado: **geoBoundaries,
Honduras**:

- ADM2: **299 polígonos** (año 2019, fuente ICF, licencia CC BY 4.0). Honduras tiene 298
  municipios oficiales: hay que revisar qué es el polígono de más (puede ser un municipio creado
  después o un cuerpo de agua).
- ADM1: 18 departamentos (OpenStreetMap, licencia ODbL).

Ubicación → municipio: prueba de punto en polígono en memoria, sin llamadas externas.

### 8.3 Qué significa "proyección municipal" en la práctica

- **Clima:** un municipio promedio mide ~377 km² y una celda de POWER ~3.700 km². Muchos municipios
  **comparten la misma celda**: ~299 municipios sobre ~50 celdas con tierra. Dos municipios vecinos
  en la misma celda tienen **el mismo clima** en el cálculo; sus planes solo difieren por el suelo.
  Si un municipio cruza varias celdas, se pondera por el área de intersección.
- **Suelo:** se agregan todos los píxeles de 250 m del municipio: mediana para el cálculo,
  percentiles para la heterogeneidad y el reparto por clase de textura (por ejemplo, "60 % franco
  arcilloso, 30 % arcilloso"). Limitación: el promedio municipal incluye bosque, ladera y zona
  urbana, no solo tierra de cultivo. Separarlos necesitaría un mapa de cobertura (otra fuente).
- **Altitud:** ver Q7. La elevación de la celda de POWER (1.021 m en el ejemplo) puede estar muy
  lejos de la real: a pocos kilómetros de ese punto, el modelo Copernicus DEM da 810, 1.081 y
  1.285 m.

### 8.4 Departamento

Un departamento es mucho más grande (Olancho tiene ~24.000 km² y varias celdas). Un único plan
departamental promediaría climas y suelos muy distintos. Ver Q6.

### 8.5 Consecuencia para el cálculo

Con 299 municipios, los perfiles de sitio (clima por quincena + suelo agregado) se pueden
**precalcular en el ETL**: 299 perfiles. La evaluación de rotaciones sigue siendo por pedido, porque
depende de la fecha, el pronóstico, el riego, la rotación actual y la prioridad.

---

## 9. Almacenamiento, precarga e infraestructura

### 9.1 Clima en Redis

**En la práctica:**

- **Qué se ejecuta:** un proceso de precarga (ETL) que hace un `GET` a la API diaria de POWER por
  cada celda de la caja de Honduras: **88 celdas** (8 filas de latitud × 11 columnas de longitud),
  con 13 parámetros, de 1996-01-01 a 2025-12-31. Después clasifica cada año con el índice ONI y
  calcula los resúmenes derivados (quincenas, ET0 diaria, inicio de lluvias, canícula).
- **Cuándo y cuántas veces:** **una sola vez**, antes de la demo. Después, **una vez al año** (en
  enero) para agregar el año cerrado. En operación normal son **0 llamadas por hora** a NASA.
- **Volumen:** medido para una celda: 2,18 MB y 2,0 s (12 parámetros, 30 años). Para 88 celdas:
  ~200 MB de descarga y ~3 a 4 minutos en secuencia (estimado a partir de la medición). Guardado
  en Redis en binario compacto (float32): 10.958 días × 13 parámetros × 4 bytes ≈ **570 KB por
  celda, ~50 MB en total** (cálculo, no medido).
- **Con quién compite:** servicio público de NASA; en Redis, con el suelo y los resultados (§ 9.5).
- **Qué pasa si crece:** +1 año por celda por año (~19 KB). Despreciable.

Formato: una clave por celda y versión de datos, por ejemplo `clima:v1:celda:14.5:-88.75`, con los
parámetros como arreglos binarios contiguos. Los derivados van en claves aparte
(`clima:v1:celda:...:quincenas`). El prefijo de versión permite recargar todo sin mezclar datos
viejos y nuevos.

### 9.2 Suelo en Redis

> **v4:** como la unidad es el municipio, la API ya no lee píxeles. El ETL sigue procesando los
> 96 rásteres de 250 m (misma descarga que se describe abajo), pero **guarda en Redis solo los
> agregados por municipio**: 299 municipios × 8 propiedades × 4 profundidades × unos pocos
> estadísticos ≈ **menos de 5 MB** (cálculo, no medido). Las teselas de píxeles dejan de cargarse
> en Redis; los binarios del ETL quedan en disco por si se necesitan después. Con esto, la memoria
> total de Redis baja a ~100 MB. La instancia de 4 GB (P4) sobra; la de 2 GB (12 USD al mes)
> alcanzaría. Queda a criterio del equipo.

**En la práctica:**

- **Qué se ejecuta:** un ETL **único** con GDAL (en un contenedor Docker, sin instalar nada en la
  máquina), sobre los **96 rásteres** (8 propiedades × 4 profundidades × 3 valores) del WebDAV de
  ISRIC:
  1. `gdalwarp` lee por HTTP solo los bloques del ráster que caen en Honduras (`/vsicurl/`),
     reproyecta de Homolosine a latitud/longitud y recorta con el contorno del país.
  2. `gdal_translate` exporta cada capa a binario crudo (int16), que se puede leer desde C# sin
     librerías geoespaciales.
  3. Un cargador escribe el resultado en Redis en **teselas**: una clave por tesela (por ejemplo, de
     256 × 256 píxeles), con los 96 valores de cada píxel juntos. Para leer un píxel, Redis devuelve
     solo los 192 bytes de ese píxel (`GETRANGE`), sin traer la tesela entera.
- **Cuándo y cuántas veces:** **una sola vez**. SoilGrids 2.0 es estático; solo se repite si ISRIC
  publica una versión nueva.
- **Volumen:** Honduras tiene ~1,8 millones de píxeles de 250 m en tierra. Con 96 valores de 2
  bytes son **~345 MB**; con el mar y los países vecinos que caen dentro de las teselas del borde,
  **~450 a 550 MB** en Redis (cálculo, no medido). La descarga es del mismo orden, una vez. Hay que
  medirlo con una sola capa antes de lanzar las 96 (tarea 1.5, § 11).
- **Con quién compite:** el servidor de archivos de ISRIC, público y compartido (solo durante la
  precarga). En Redis, es el bloque más grande de memoria.
- **Qué pasa si crece:** no crece.

Alternativa para reducir memoria: usar el producto **agregado a 1 km** de SoilGrids (16 veces menos
píxeles, ~35 MB). Se pierde detalle dentro de la finca, pero sigue siendo 3.700 veces más fino que
el clima. Ver P4 (§ 12).

### 9.3 Pronóstico en Redis

> **v4, ECMWF SEAS5 vía Open-Meteo:**
>
> - **Qué se ejecuta:** un `GET` a `seasonal-api.open-meteo.com/v1/seasonal` por cada celda de
>   POWER con tierra (~50 a 88), pidiendo lluvia y temperatura diarias o anomalías mensuales, a
>   7 meses.
> - **Cuándo y cuántas veces:** **una vez al mes**, después de la corrida del día 5. ~88 llamadas
>   al mes, o sea ~0,12 por hora en promedio, todas juntas en una ráfaga mensual.
> - **Volumen medido:** para un punto, con 2 variables diarias a 183 días y los 51 miembros:
>   **103 KB en 1,4 s**. Para 88 celdas: ~9 MB y ~2 minutos al mes. En Redis, solo las
>   probabilidades por tercil por celda y mes: kilobytes.
> - **Con quién compite:** API pública de Open-Meteo, gratuita para **uso no comercial**. Si el
>   proyecto pasa a uso comercial, hace falta un plan pago. Atribución obligatoria a ECMWF.
> - **Qué pasa si crece:** no crece con los usuarios, porque se baja por celda y no por pedido.
> - **Fuente no NASA:** es la tercera fuente del sistema, aprobada en Q3. Hay que declararla en
>   cada respuesta igual que POWER y SoilGrids.
> - **Ventaja adicional:** su grilla es de 36 km, más fina que la de POWER (~55 × 67 km).
>
> Lo que sigue de esta sección (ONI e IRI) queda solo como referencia.

**En la práctica:**

- **Qué se ejecuta:** un `GET` al archivo de texto del ONI de NOAA CPC (~10 KB) y uno a la fuente
  del pronóstico ENSO (IRI o CPC). Fase 2: descarga de las probabilidades por tercil de NMME.
- **Cuándo y cuántas veces:** **una vez al mes** (las fuentes se actualizan mensualmente), con una
  tarea programada. Equivale a ~0,003 llamadas por hora.
- **Volumen:** kilobytes. Fase 2 (NMME): no medido.
- **Con quién compite:** servicios públicos de NOAA e IRI.
- **Qué pasa si crece:** no crece.

Cuando cambia el pronóstico, cambia su versión (`pronostico:v2026-11`) y los resultados viejos dejan
de usarse, porque su clave incluye esa versión (§ 9.4).

### 9.4 Resultados de cálculo en Redis

- Cada evaluación se guarda con una clave derivada de **todo lo que la determina**: celda, píxel o
  polígono, altitud, riego, rotación actual, prioridad, trimestre de inicio, versión de la ficha de
  cultivos y versión del pronóstico. Si cualquiera de esas cosas cambia, la clave cambia.
- Vencimiento (TTL) de 35 días, para que no se acumulen resultados de pronósticos viejos.
- Explicaciones (D7): **plantillas**. Ver § 9.6.

### 9.5 Configuración de Redis y memoria

- `maxmemory-policy noeviction`: Redis **no** borra datos por su cuenta. Si se llena, las escrituras
  de resultados fallan y la API sigue respondiendo sin guardar (el clima y el suelo nunca se borran).
- Persistencia con snapshot (RDB) después de cada precarga. Como el clima y el suelo **se pueden
  reconstruir** con el ETL, Redis no es la única copia: los archivos binarios del ETL se guardan en
  el disco de la instancia y en un snapshot de Lightsail.
- Memoria estimada en Redis: clima ~50 MB + suelo ~450 a 550 MB + resultados y pronóstico (menos de
  100 MB) ≈ **600 a 700 MB**.

### 9.6 Las explicaciones (D7) en simple

Cada recomendación trae frases como "En 4 de 30 años el maíz sembrado en esta ventana habría perdido
más de la mitad del rendimiento por falta de agua". La duda es **quién escribe esas frases**:

- **Plantillas (propuesta):** frases fijas con huecos que se llenan con los números calculados
  ("En {n} de {N} años el {cultivo}..."). Siempre dicen la verdad, porque solo pueden decir lo que
  calculó el motor. Suenan algo repetitivas.
- **Un modelo de lenguaje (LLM):** recibe los números y redacta un texto más natural. Puede
  **inventar** cifras, causas o consejos que el motor nunca calculó, y en una herramienta agrícola
  sin agrónomo que revise eso es peligroso. Además agrega una API externa, costo por consulta y
  latencia.

Propuesta: plantillas en el MVP. Si después se quiere un texto más natural, el LLM solo **reescribe**
lo que dicen las plantillas, y un validador rechaza el texto si contiene un número que no está en el
resultado.

### 9.7 Supabase

Por definir (P2, § 12). Propuesta: Supabase guarda solo lo que tiene que **durar y pertenecer a
alguien**: usuarios, parcelas guardadas e historial de planes generados. No guarda datos de clima,
suelo ni cálculos.

Riesgos verificados del plan gratuito (fuentes de terceros, confirmar en supabase.com/pricing): 500
MB de base de datos, y el proyecto **se pausa después de 7 días sin actividad**; reactivarlo es
manual. Si la demo cae después de una semana sin uso, la API falla en lo que dependa de Supabase.
Mitigación: un ping periódico, o que la API funcione sin Supabase para las consultas anónimas.

### 9.8 Lightsail

Planes Linux con IPv4 (documentación oficial de AWS, consultada el 2026-11-14):

| Plan | RAM | vCPU | Disco | USD/mes |
| ---- | --- | ---- | ----- | ------- |
| Small | 2 GB | 2 | 60 GB | 12 |
| **Medium (propuesta)** | **4 GB** | 2 | 80 GB | **24** |
| Large | 8 GB | 2 | 160 GB | 44 |

Presupuesto de memoria en una sola instancia: Redis ~0,7 GB + margen para el snapshot (Redis copia
memoria al guardar) + API .NET ~0,3 a 0,5 GB + sistema operativo y proxy ~0,4 GB. **2 GB queda
justo**; 4 GB deja margen. Con suelo a 1 km (P4), 2 GB alcanza.

Los planes de Lightsail son de **CPU con ráfagas** (burstable): uso sostenido alto de CPU agota los
créditos. Las simulaciones son cortas, pero hay que medir la carga de la búsqueda de rotaciones.

Despliegue propuesto: Docker Compose con tres servicios (API .NET 10, Redis y un proxy con HTTPS,
por ejemplo Caddy). El ETL corre como contenedor aparte, a demanda.

**Seguridad (obligatorio):**

- Redis escucha **solo en la red interna de Docker o en 127.0.0.1**, con contraseña. El puerto
  6379 **nunca** se abre en el firewall de Lightsail. Un Redis expuesto a internet se compromete en
  horas.
- El firewall de Lightsail abre solo 80 y 443, y el 22 restringido a las IP del equipo.
- Las credenciales de Supabase y de Redis van en variables de entorno de la instancia, nunca en el
  repositorio.

---

## 10. Puntos de falla y brechas conocidas

### 10.1 Datos de POWER con valores anómalos (crítico, verificado)

Lluvia de enero a marzo en el punto de ejemplo:

| Año | POWER (mm) | ERA5 (mm) |
| --- | ---------- | --------- |
| 2020 | 70  | 20 |
| 2021 | **671** | 41 |
| 2022 | **570** | 63 |
| 2023 | **269** | 38 |
| 2024 | **220** | sin consultar |

Entre 1995 y 2020 POWER nunca pasó de 104 mm en ese trimestre, y la fuente declarada es MERRA-2 en
todos los años. Causa sin determinar. ERA5 se usó **solo para diagnosticar** este problema; por la
decisión P6 no entra al sistema.

Mitigación (v3, solo con POWER):

- El ETL detecta valores atípicos con el **propio histórico de la celda**: por mes calendario, una
  lluvia mensual por encima de Q3 + 3 × RIC (rango intercuartílico) de los 30 años se marca.
  Aplicado al total de enero a marzo del punto de ejemplo (calculado a mano sobre los 30 valores),
  el umbral queda en ~236 mm: se marcan 2021 (671), 2022 (570) y 2023 (269), pero **2024 (220) no**.
  El umbral exacto se fija con los datos mensuales en la fase 1.
- El mes marcado se **excluye** de los cálculos de ese mes en esa celda (la muestra baja, por
  ejemplo, de 30 a 26 años) y se reporta en `dataQuality`.
- Se usan percentiles en vez de promedios.
- Limitación: un valor anómalo que no sea extremo frente al propio histórico no se detecta. Sin una
  segunda fuente de lluvia no hay forma de confirmarlo.

### 10.2 Canícula subestimada por el sesgo de llovizna (alto, verificado)

Ver § 7.5.

### 10.3 Pronóstico estacional débil justo en la primera (alto)

La habilidad de los pronósticos es menor en mayo-julio que en agosto-octubre (§ 2.4). La ponderación
por pronóstico de S1 tiene que mostrar su nivel de confianza por temporada y no presentarse como
certeza.

### 10.4 Resolución gruesa en montaña (alto)

Corrección por altitud (§ 8.4), CHIRPS en fase 2 y resolución declarada en cada respuesta.

### 10.5 Ficha de cultivos sin validación agronómica (alto)

Ver § 5.5 y § 5.6.

### 10.6 Sin precios ni costos (medio)

No hay API de precios para Honduras. El subpuntaje de ingreso usa una tabla estática con fecha y
fuente (por ejemplo, precios de referencia de la SAG o de mercados mayoristas), y se declara en
`missingInformation`. Lo que sí se calcula es el **ahorro de fertilizante nitrogenado** y los
**milímetros de riego** necesarios.

### 10.7 Precarga de suelo no probada (medio)

El ETL desde WebDAV y la reproyección no se probaron todavía. Volumen y tiempo estimados, no
medidos. Tarea 1.5 (§ 11) lo mide con una sola capa primero.

### 10.8 Un solo servidor (medio)

API y Redis en la misma instancia: si la instancia cae, cae todo. Aceptable para la demo; mitigado
con el snapshot de Lightsail y el ETL reproducible.

### 10.9 Riesgo de "parecer agrónomo" (medio)

Lenguaje de probabilidades, disclaimer en cada respuesta y salida pensada para el técnico.

---

## 11. Fases de implementación

### Fase 0: fundaciones (sin red)

1. Solución .NET 10 con cuatro capas (Domain, Application, Infrastructure, Presentation) y tests.
2. Motores de dominio con tests contra ejemplos resueltos de FAO-56 (capítulo 3) y valores
   publicados de Saxton y Rawls.
3. Ficha de cultivos en JSON con su esquema y validador (fuente y confianza obligatorias).

### Fase 1: precarga y Redis

1. Docker Compose local con Redis.
2. ETL de POWER: 88 celdas, control de calidad, clasificación ENSO, derivados.
3. Test de la regla de celda (§ 8.2).
4. Prueba de que el control de calidad marca 2021–2023 en el punto de ejemplo.
5. ETL de SoilGrids: **primero una sola capa** para medir tiempo, volumen y memoria; después las 96.
6. Actualización mensual del pronóstico ENSO.

### Fase 2: motor de recomendación (MVP)

7. Perfil del sitio: quincenas, inicio de lluvias, canícula por déficit, ADT, aptitud.
8. Generador de rotaciones y simulación sobre 30 años con ponderación por pronóstico.
   **Medir** el tiempo con el catálogo completo.
9. Indicadores de suelo, puntaje, plantillas de explicación.
10. Agregación por trimestre, semestre y temporada.
11. Endpoints, ProblemDetails, OpenAPI y colección de Postman con los casos de demo.
12. Pruebas de aceptación agronómica (§ 5.6).

### Fase 3: despliegue

13. Instancia de Lightsail, firewall, Docker Compose, HTTPS y variables de entorno.
14. Cargar Redis con los datos del ETL y hacer el snapshot.
15. Integración con Supabase (según P2).

### Fase 4: mejoras

16. Ponderación por terciles (NMME / NASA GEOS-S2S).
17. CHIRPS para lluvia y canícula; DEM para altitud y pendiente.
18. Coberturas y cultivos con confianza C, a medida que aparezcan fuentes.

### Criterio de "listo" del MVP

- Para lat 14.30, lon -88.60 la API devuelve ≥ 3 rotaciones, incluida la actual, por trimestre,
  semestre y temporada, con probabilidad de pérdida, indicadores de suelo, fuentes, nivel de
  confianza y advertencias de calidad.
- Pasan todas las pruebas de aceptación agronómica de § 5.6.
- Ninguna consulta de usuario llama a NASA ni a ISRIC.
- Todos los motores de dominio tienen tests con valores de referencia publicados.

---

## 12. Preguntas abiertas

### 12.0 Dudas restantes (v4, respondidas en § 0.3)

**Q6. Departamento.** Si el usuario elige un departamento, ¿qué devuelve la API?

- (a) Un plan por cada municipio del departamento (el frontend muestra un mapa o una lista).
  Propuesta.
- (b) Un único plan departamental promediado. Mezcla climas y suelos distintos: la recomendación
  puede no servir para ningún municipio en particular.

**Q7. Altitud del municipio.** Sin altitud, no se puede corregir la temperatura, y eso afecta qué
cultivos son viables (papa, frijol de altura). Opciones:

- (a) Precalcular, **una sola vez** en el ETL, la altitud de cada municipio (mediana y rango) con
  el modelo de elevación Copernicus DEM vía la API de elevación de Open-Meteo. Es la misma
  plataforma que el pronóstico y acepta 100 puntos por llamada. Muestreando cada ~5 km son ~4.500
  puntos, ~45 llamadas, una vez. (Cálculo, no medido.)
- (b) No corregir: usar la elevación de la celda de POWER, con una advertencia.

Propuesta: (a).

**Q8. Lo observado en el año en curso.** Para que el plan reaccione a cómo vienen las lluvias
**este año** (por ejemplo, "mayo vino seco y el inicio se atrasó"), habría que bajar de POWER el año
en curso, por celda, cada semana: ~88 llamadas por semana, unos pocos KB cada una. Advertencia: en
los datos recientes POWER declara otras fuentes (GEOS-IT, en la respuesta que pegaste para noviembre
de 2026) que en el histórico (MERRA-2). Compararlos puede introducir sesgo; hay que medirlo antes.
Propuesta: **fase 2**, no MVP. El MVP actualiza el plan por la fecha y por el pronóstico mensual.

**Q9. Stack (Q5).** Sigue pendiente y **bloquea el inicio del código**: la fase 0 es crear la
solución.

### 12.1 Dudas restantes (v3, respondidas en § 0.2)

**Q1. Qué dice cada trimestre.** El plan es una **proyección**: dice qué *se recomienda* sembrar, qué
*estaría* en campo y qué *se cosecharía* en cada trimestre si el productor sigue la rotación. Eso sale
del cálculo; **no requiere guardar** los cultivos del productor. Lo único que la API necesita saber
es qué tiene sembrado hoy y su rotación habitual, y lo recibe **en el mismo pedido**, sin guardarlo.
El reto exige comparar contra la rotación actual. ¿Está bien así, o prefieren que cada trimestre
diga solo qué sembrar?

**Q2. Sin Supabase en el MVP.** Si todo va en Redis y no se guardan datos del productor, la API no
necesita usuarios. Propuesta: API **pública, sin login**, con límite de pedidos por IP para proteger
la instancia. ¿De acuerdo?

**Q3. Pronóstico de 6 meses y "solo 2 fuentes".** NASA POWER y SoilGrids **no tienen pronóstico**:
POWER solo trae el pasado. Para ponderar los próximos 6 meses hace falta **una tercera fuente**:

- (a) El estado de **El Niño / La Niña** (NOAA o IRI): un archivo de texto al mes. Simple.
- (b) Mapas de **probabilidad de lluvia** de los modelos NASA (NMME, que incluye NASA GEOS-S2S): más
  "dato NASA", pero hay que procesar archivos científicos (NetCDF). Más trabajo.
- (c) **Ninguna**: los 12 meses usan solo la climatología de 30 años, sin pronóstico.

¿Cuál? Si "solo 2 fuentes" es una regla estricta, la respuesta es (c).

**Q4. Explicaciones.** El reto exige explicar cada recomendación. "De momento no" a las plantillas
¿significa que la API **no devuelve texto**? Propuesta: la API devuelve las razones **estructuradas**
(código de la razón, métrica, valor y fuente) y el frontend arma la frase. Así se cumple el reto sin
plantillas en la API.

**Q5. Stack.** ¿Confirmamos **.NET 10 (C#)** para la API? El equipo tiene que poder mantenerlo.

**Q6. Supuestos de entrada sin una tercera fuente** (confirmar, no hace falta decidir nada nuevo):

- La **altitud** de la parcela la declara el usuario (opcional). Si no la da, se usa la elevación de
  la celda de POWER, con una advertencia.
- La **pendiente** la declara el usuario como clase (plano, ondulado, ladera).
- El **contorno de Honduras** para validar coordenadas y recortar el suelo es un archivo estático de
  límites administrativos versionado en el repositorio, no una API.
- Los cultivos con valores críticos de baja confianza (ciclo, Kc, raíz o tolerancia a la sequía) se
  muestran como **exploratorios**.

### 12.2 Preguntas de v2 (ya respondidas, ver § 0.1)

**P1. Trimestre de inicio.** ¿Los trimestres son de calendario (ene–mar, abr–jun, jul–sep,
oct–dic) o se cuentan desde el mes en que se pide el plan? Y cuando un cultivo cruza dos trimestres
(maíz de mayo a septiembre), ¿basta con mostrar en cada trimestre qué se siembra, qué está en campo y
qué se cosecha (propuesta § 6.2), o se esperaba **un cultivo por trimestre**? Esto último obligaría
a usar solo cultivos de ~90 días o a cortar ciclos, y agronómicamente no es correcto.

**P2. Supabase.** ¿Qué va ahí exactamente: usuarios y login, parcelas guardadas, historial de
planes? ¿La API necesita autenticación o es pública para la demo?

**P3. Riego.** ¿Incluimos parcelas con riego? Si sí, el primer trimestre se abre a melón, sandía,
hortalizas y arroz, y el sistema reporta los milímetros de riego necesarios. No sabemos cuánta agua
tiene disponible cada productor; el riego sería un dato que declara el usuario.

**P4. Resolución del suelo y presupuesto.** 250 m (~0,5 GB en Redis, instancia de 4 GB, 24 USD al
mes) o 1 km (~35 MB, alcanza una de 2 GB, 12 USD al mes).

**P5. Fuente del pronóstico.** MVP con ponderación por ENSO (simple, un archivo de texto al mes) y
terciles NMME / NASA GEOS-S2S en fase 2. ¿O quieren los terciles desde el MVP para tener más "dato
NASA"? Implica procesar NetCDF.

**P6. Años anómalos de POWER.** Cuando el control de calidad marca un mes (como enero a marzo de
2021), ¿se **excluye** ese año del cálculo de ese mes, o se **reemplaza** la lluvia con otra fuente
(ERA5 o CHIRPS)? Excluir es simple, pero reduce la muestra; reemplazar mezcla fuentes y hay que
declararlo.

**P7. Explicaciones (D7).** Con la explicación de § 9.6, ¿confirmamos plantillas para el MVP?

**P8. Revisión externa.** ¿Alguien del equipo puede contactar a DICTA, Zamorano, UNAG o FAO Honduras
para una revisión corta de la ficha de cultivos? Si no, todo lo que no sea confianza A sale marcado
como baja confianza.

### Si se aprueba este plan, lo que va a pasar es

Antes de la demo corremos una precarga **una sola vez**:

- Se bajan 30 años de clima de NASA para 88 celdas que cubren Honduras.
- Se bajan los mapas de suelo de ISRIC recortados al país.
- Todo queda en un Redis dentro de la misma máquina de Lightsail, que nunca se expone a internet.

Una vez al mes, una tarea baja el pronóstico de El Niño o La Niña. Cuando alguien pide un plan, la
API **no llama a NASA ni a ISRIC**: lee de Redis, simula cada rotación en los 30 años (con más peso
a los años parecidos al pronóstico para los primeros 6 meses) y devuelve el plan por trimestre y
semestre con sus probabilidades. El resultado queda guardado en Redis 35 días.

---

## 13. Referencias

- Reto Field Shift (starter): https://github.com/GDriem/space-apps-starter/blob/main/retos/07-field-shift.md
- Reto oficial: https://www.spaceappschallenge.org/2026/challenges/adapting-farms-with-nasa-data/
- NASA POWER, metodología: https://power.larc.nasa.gov/docs/methodology/
- NASA POWER, API diaria: https://power.larc.nasa.gov/docs/services/api/temporal/daily/
- Límite de uso de POWER (foro Earthdata): https://forum.earthdata.nasa.gov/viewtopic.php?p=15146
- SoilGrids, FAQ: https://docs.isric.org/globaldata/soilgrids/SoilGrids_faqs_01.html
- SoilGrids, acceso y uso justo: https://docs.isric.org/globaldata/soilgrids/SoilGrids_faqs_02.html
- SoilGrids WebDAV: https://files.isric.org/soilgrids/latest/data/
- API REST de ISRIC: https://rest.isric.org/
- FAO-56, Tablas 11 y 12: https://www.fao.org/4/x0490e/x0490e0b.htm
- FAO-56, Tabla 22: https://www.fao.org/4/x0490e/x0490e0e.htm
- FAO-33 (Doorenbos y Kassam, 1979), Yield response to water.
- Saxton, K. E. y Rawls, W. J. (2006). Soil Science Society of America Journal 70: 1569–1578.
- Renard, K. G. et al. (1997). RUSLE, USDA Agriculture Handbook 703.
- Williams, J. R. (1995). The EPIC model (factor K).
- Arnoldus, H. M. J. (1980). Índice de Fournier modificado.
- Ecocrop (parámetros, opinión experta): https://rdrr.io/cran/Recocrop/man/parameters.html
- NOAA CPC, índice ONI: https://www.cpc.ncep.noaa.gov/data/indices/oni.ascii.txt
- IRI, pronóstico ENSO: https://iri.columbia.edu/our-expertise/climate/forecasts/enso/current/
- Comparación de pronósticos estacionales C3S y NMME en Centroamérica: https://www.ncbi.nlm.nih.gov/pmc/articles/PMC11921286/
- Habilidad por temporada en el Corredor Seco (EGU22, Kowal et al.): https://presentations.copernicus.org/EGU22/EGU22-367_presentation-h688076.pdf
- Evaluación de NMME en Centroamérica (NOAA CPC): https://vlab.noaa.gov/documents/17693964/39260487/Recalde_Cristina_2024-s2s.pdf
- Perspectiva del Clima de América Central (CRRH-SICA), ejemplo: https://www.imhpa.gob.pa/uploads/documentos/lxxvii_perspectiva_del_clima_2325_de_abril_2025.pdf
- Planes de Lightsail: https://docs.aws.amazon.com/lightsail/latest/userguide/amazon-lightsail-bundles.html
- Límites del plan gratuito de Supabase (terceros): https://www.jetadmin.io/blog/supabase-pricing-2026-guide-to-plans-limits-and-real-world-costs/
- Calendario de primera y postrera: https://www.elheraldo.hn/honduras/cuando-deben-comenzar-sembrar-los-productores-ante-llegada-el-nino-CP30486172
- Apante en el Litoral Atlántico: https://www.bloomberglinea.com/2022/10/24/honduras-recurre-al-frijol-nicaraguense-para-hacer-frente-a-la-especulacion-de-precios/
- Variedades DICTA y Zamorano (contexto): https://bdigital.zamorano.edu/bitstreams/706cafa5-7a3d-47a6-920a-4146fcd123be/download
- ERA5 vía Open-Meteo (validación cruzada): https://open-meteo.com/en/docs/historical-weather-api
