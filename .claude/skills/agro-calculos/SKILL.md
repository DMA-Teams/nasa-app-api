---
name: agro-calculos
description: "Reglas para implementar y probar los cálculos científicos de nasa-app-api: ET0 Penman-Monteith y Hargreaves (FAO-56), curva de Kc, pedotransferencia de Saxton-Rawls, balance hídrico diario, respuesta del rendimiento al agua (FAO-33), inicio de lluvias, canícula por déficit, grados-día, aptitud tipo Ecocrop, RUSLE, balance de nitrógeno, ponderación de años por pronóstico y puntaje de rotaciones. Usar al escribir, revisar o depurar cualquier código de internal/agro o internal/catalog, la ficha de cultivos en JSON, o los tests de esas fórmulas."
paths:
  - "internal/agro/**"
  - "internal/catalog/**"
  - "**/testdata/**"
---

# Cálculos agronómicos

El valor de este proyecto está en que los números sean **correctos y rastreables**. Un error de
unidades en una fórmula produce recomendaciones plausibles y falsas, que nadie detecta porque no hay
un agrónomo en el equipo. Por eso estas reglas son más estrictas que las del resto del código.

Base científica y valores de referencia: `docs/plan-implementacion.md` § 5 y § 7.

## Reglas

1. **Dominio puro.** `internal/agro` no importa nada de I/O: ni `net/http`, ni Redis, ni `os`, ni
   `time.Now()`. La fecha, el clima, el suelo y la ficha entran como parámetros. Mismo input, mismo
   output, siempre.
2. **Unidades en el tipo.** Cada magnitud física tiene un tipo con nombre (`Millimeters`,
   `MegajoulesPerSquareMeterDay`, `Celsius`, `Kilopascals`, `MetersPerSecond`, `Fraction`). Las
   conversiones son funciones explícitas con su test (por ejemplo, energía de evapotranspiración a
   milímetros: dividir por 2,45 MJ/kg). Nunca sumar dos tipos distintos sin convertir.
3. **Valores de relleno.** `-999` de POWER es **dato faltante**, no un número. Se convierte a
   "faltante" en el adaptador y el dominio nunca lo ve como valor.
4. **Cada fórmula tiene un test contra un ejemplo publicado.** El nombre del test dice la fuente:
   `TestET0PenmanMonteith_FAO56Example18`, `TestSaxtonRawls_Table3Loam`. La tolerancia se fija según
   la precisión del ejemplo publicado (por ejemplo, ±0,1 mm/día si FAO-56 da un decimal).
5. **Ningún número mágico.** Las constantes físicas y agronómicas son constantes con nombre. Los
   parámetros de los cultivos viven en la ficha JSON, con `source` y `confidence` obligatorios
   (A, B o C, ver plan § 5.5). El origen de las constantes del código se documenta en
   `docs/fuentes.md`.
6. **Invariantes probadas.** Agotamiento entre 0 y el agua total disponible; Ks entre 0 y 1; pesos
   de años que suman 1; probabilidades entre 0 y 1; rendimiento relativo nunca negativo. Se prueban
   con tests de propiedades (`testing.F` o tablas con casos límite), no solo con un ejemplo feliz.
7. **Precisión.** Cálculo en `float64`. `float32` solo para almacenamiento en Redis. Redondear
   únicamente al presentar, nunca en pasos intermedios.
8. **Trazabilidad en la salida.** Cada resultado lleva las versiones de datos, ficha y pronóstico
   que lo produjeron, y las razones estructuradas (código, métrica, valor, fuente). El texto lo arma
   el frontend.
9. **Aceptación agronómica.** Los casos de plan § 5.6 (sin secano en enero–marzo en el Corredor
   Seco, frijol de postrera en el occidente, apante en el Litoral Atlántico, etc.) son tests de
   aceptación con datos grabados. Si uno falla, el motor está mal aunque los unitarios pasen.

## Rendimiento del motor

- La simulación diaria es el ciclo caliente: slices preasignados por simulación, sin mapas ni
  interfaces dentro del ciclo, sin asignaciones por día.
- Benchmarks con `b.Loop()` (Go 1.24+) para el balance hídrico y para la evaluación de un municipio
  completo. Medir antes de optimizar (`golang-benchmark`, `golang-performance`).

## Fuentes

- FAO-56: https://www.fao.org/4/x0490e/x0490e00.htm
- FAO-33 (Doorenbos y Kassam, 1979), Yield response to water.
- Saxton y Rawls (2006), Soil Sci. Soc. Am. J. 70: 1569–1578.
- RUSLE, USDA Agriculture Handbook 703.
