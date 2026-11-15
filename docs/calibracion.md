# Plan de calibración del motor

> Estado: **etapa 1 implementada, pendiente de revisión** (2026-11-15). Etapa 2 sin empezar.
> Escrito el 2026-11-15, después de la primera prueba real con los datos de Lempira.

## 0. Resultado de la etapa 1 (2026-11-15)

Medido con `internal/plan/lempira_acceptance_test.go` sobre los datos reales grabados en
`testdata/lempira/` (28 municipios, inicio 2026-11-15, rotación actual maíz y frijol):

- Ingreso eliminado; pesos agua/suelo según § 1.
- Frijol en relevo dentro del maíz en madurez: pérdida del frijol < 30 % en los 28 municipios.
- **Decidido por el usuario (2026-11-15):** las fechas de siembra salen de las fórmulas, no de una
  ventana de calendario fija. Se quitó la prueba "frijol entre el 15 de agosto y el 30 de
  septiembre". Tambla siembra el frijol en relevo el 1 de agosto porque su maíz, más caliente,
  madura antes; la simulación le da pérdida < 30 % y eso es lo que se valida.
- Coberturas en relevo solo sobre maíz o sorgo; sin siembras de secano en enero–marzo.
- **Hallazgo:** sin el ingreso, la cobertura sola (mucuna) ganaba en todos los municipios. Se agregó
  la regla "toda rotación generada lleva al menos un cultivo de cosecha" (confirmada por el usuario
  el 2026-11-15).
  Con ella gana "sorgo → mucuna en relevo" en 23 municipios y "maíz → mucuna en relevo" en 5.
- Las fechas de siembra candidatas pasaron a ser el inicio de cada quincena de calendario (1 y 16).
- Ciclo del maíz acortado a 112 días a 24 °C (confianza C), ver `docs/fuentes.md`.
- Pendiente: reconstruir la API en WSL y comparar con `data/smoke/antes/` usando
  `node data/smoke/summary.js antes` y `node data/smoke/summary.js`.

## 1. Decisión de alcance (2026-11-15)

La app recomienda cultivos y rotaciones **solo a partir de condiciones climáticas y de suelo**.
Quedan **fuera del alcance**: precios, costos, insumos, plagas y enfermedades, mercados, rendimientos
oficiales y datos de variedades comerciales.

Consecuencias en el código (a implementar):

1. **Eliminar el subpuntaje de ingreso.** El puntaje total pasa a ser agua + suelo.
   - Se quitan `relativeValue` de la ficha, `incomeScore` de `internal/agro/score.go` y la prioridad
     `income` (API, OpenAPI y validación). Quedan `balanced`, `water` y `soil`.
   - Pesos propuestos: balanced 0,55 agua / 0,45 suelo; water 0,75 / 0,25; soil 0,30 / 0,70.
     Son supuestos C y se documentan en `docs/fuentes.md`.
2. `referenceYieldTHa` se mantiene **solo** para el balance de nitrógeno, que es una propiedad del
   suelo. No se usa para ingreso.
3. Se descarta la penalización por riesgo sanitario que se había propuesto para las hortalizas: es
   agronomía de plagas, fuera del alcance.

## 2. Problemas detectados en la prueba real (San Andrés, Lempira)

| # | Problema | Causa raíz | ¿Se resuelve con datos de NASA? |
| - | -------- | ---------- | ------------------------------- |
| 1 | "Chile dulce → mucuna" gana en 22 de 28 municipios | El índice de ingreso (sin fuente) inflaba las hortalizas; en lluvias el agua no las limita | **Ya no aplica en su mayor parte**: se elimina el ingreso. Hay que **volver a medir** después del cambio |
| 2 | Frijol de postrera con 61 % de pérdida, sembrado el 11 de octubre | Ciclo del maíz demasiado largo a ~1.000 m (supuesto: 24 °C de referencia) y sin relevo de frijol dentro del maíz | **Sí, en parte**: la fenología satelital (NDVI) mide las fechas reales de siembra y cosecha |
| 3 | Canícula detectada en solo 11 de 30 años | Umbral no calibrado y sesgo de llovizna de MERRA-2 | **Sí**: GPM IMERG (lluvia a ~10 km) y SMAP (humedad del suelo) |
| 4 | Mucuna en relevo sobre chile dulce | La regla permite relevo de cobertura sobre cualquier cultivo de cosecha | No hace falta: es una regla |

## 3. Etapas

### Etapa 1: reglas y alcance (sin datos nuevos)

1. Eliminar el ingreso (§ 1) en el motor, la ficha (`tools/gencatalog/gen.js`), la API, el OpenAPI
   (`api/openapi.yaml` + `tools/genopenapi/gen.js`) y las pruebas.
2. Relevo de coberturas solo sobre cultivos de cosecha Poaceae (maíz, sorgo).
3. Relevo de frijol dentro del maíz: permitir sembrar frijol (y otras leguminosas de grano marcadas
   en la ficha) cuando el maíz entra en la etapa final (maduración, "doblado").
4. Ajuste provisional del ciclo del maíz contra los calendarios publicados de primera (siembra
   mayo–junio, cosecha en septiembre) y postrera (siembra agosto–septiembre).
5. Volver a correr Lempira en WSL y comparar antes y después (`data/smoke/summary.js`).
6. **Pruebas de aceptación con datos reales** (respuestas de Lempira grabadas en `testdata/`):
   - maíz–frijol viable con pérdida del frijol < 30 %;
   - ~~frijol de postrera sembrado entre el 15 de agosto y el 30 de septiembre~~ (descartada: las
     fechas las deciden las fórmulas, no una ventana fija);
   - sin coberturas en relevo sobre hortalizas;
   - sin siembras de secano en enero–marzo.

### Etapa 2: calibración con datos de NASA (cuenta Earthdata disponible)

Los datos se usan **solo para calibrar, offline y una vez**. La API en producción no los consulta.

| Dato | Uso | Producto y acceso |
| ---- | --- | ----------------- |
| GPM IMERG diario (~0,1°, 2000 en adelante) | Umbral de la canícula; contraste de la lluvia de POWER | GES DISC, recorte a Lempira |
| SMAP humedad del suelo (~9 km, 2015 en adelante) | Comprobar que el suelo se seca en la canícula detectada y que el balance hídrico simulado tiene sentido | AppEEARS |
| MODIS NDVI MOD13Q1 (250 m, 16 días, 2000 en adelante) | Fechas reales de siembra, máximo y cosecha en zonas de cultivo; años malos (NDVI bajo) | AppEEARS |
| MODIS Land Cover MCD12Q1 (500 m) | Máscara de cultivos, para no medir bosque | AppEEARS |

Pasos:

1. **Canícula.** Detectarla con IMERG y POWER en 2000–2025 y elegir el umbral que reproduce los años
   fuertes sin falsas alarmas en los normales. La lista de años fuertes de referencia (2015, 2018,
   2019 y 2023 según reportes de prensa y de FEWS NET) **hay que confirmarla con IMERG y SMAP**.
   Ajustar con unos años y comprobar con otros.
2. **Ciclo y fechas.** Con NDVI sobre píxeles de cultivo de Lempira, obtener la fecha típica de
   reverdecimiento (≈ siembra + 2 a 3 semanas), el máximo y el secado (≈ cosecha). Ajustar la
   temperatura de referencia o las etapas del maíz y del frijol hasta que coincidan.
3. **Comprobación histórica.** Simular 2014–2024 y comprobar que los años con pérdida simulada alta
   coinciden con NDVI máximo bajo y con SMAP seco. Es la evidencia principal de validez del motor
   ante el reto.
4. Registrar cada parámetro calibrado en `docs/fuentes.md`, con su confianza subida a B cuando quede
   respaldado por los datos.

**Decidido (2026-11-15):** IMERG se usa **solo para calibrar**. No es fuente de la API en
producción.

En la práctica (estimaciones **no medidas**):

- **IMERG:** descarga única recortada a Lempira, 2000–2025: unas 9.500 imágenes diarias, decenas de
  MB en total.
- **NDVI, SMAP y cobertura por AppEEARS:** un pedido por área de Lempira, 2014–2025. Es asíncrono
  (minutos a horas del lado de NASA) y pesa decenas a cientos de MB.
- **Dónde corre:** en la máquina de desarrollo, una sola vez. No toca Redis de producción ni la API.

### Etapa 3: resto del país

Repetir la comprobación en dos o tres departamentos con climas distintos: Litoral Atlántico
(Atlántida, por el apante), Pacífico seco (Choluteca) y altiplano (Intibucá). Así se confirma que la
calibración de Lempira no se sobreajustó a un solo lugar.

## 4. Credenciales de NASA Earthdata

- El usuario ya tiene la cuenta y el token (2026-11-15).
- El token **no** va en el repositorio, ni en el chat, ni en archivos versionados. Se guarda como
  variable de entorno del usuario, `EARTHDATA_TOKEN`, o en un archivo fuera del repositorio.
- Los tokens de Earthdata vencen; revisar la fecha de vencimiento en la cuenta antes de la etapa 2.
- Antes de programar, verificar cómo se autentica cada servicio: GES DISC acepta el token como
  `Authorization: Bearer`, y AppEEARS usa su propio login con las credenciales de Earthdata.
  Confirmarlo en su documentación.
