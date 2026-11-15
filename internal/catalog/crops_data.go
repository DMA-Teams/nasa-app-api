package catalog

const cropsJSON = `{
  "version": "2026-11-15",
  "crops": [
    {
      "id": "maize",
      "name": "Maíz",
      "family": "Poaceae",
      "role": "harvest",
      "stages": {
        "value": [
          18,
          31,
          36,
          27
        ],
        "source": "FAO-56 Tabla 11 (Nigeria, húmedo, 125 días) con las mismas proporciones, acortado a 112 días para que a ~1.000 m (San Andrés, Lempira) la primera sembrada a mediados de mayo se coseche a fines de septiembre, como en los calendarios de primera (siembra mayo-junio, cosecha septiembre); ajuste provisional, a calibrar con NDVI (docs/calibracion.md, etapa 2)",
        "confidence": "C"
      },
      "kc": {
        "value": [
          0.3,
          1.2,
          0.35
        ],
        "source": "FAO-56 Tabla 12; Kc ini típico de cultivo de campo",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 1.2,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.55,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1.25,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "kyByPhase": {
        "value": [
          0.4,
          1.5,
          0.5,
          0.2
        ],
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación; fases asignadas a etapas FAO-56 según docs/fuentes.md",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          10,
          10,
          18,
          32,
          40
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 35,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          5.5,
          7.5,
          8.5
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          2200
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "moderate",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          16
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 1.8,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.5,
          0.25,
          0.35
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ],
      "relay": {
        "value": {
          "hosts": [
            "cover",
            "grain"
          ]
        },
        "source": "Decisión de alcance del 2026-11-15 (docs/calibracion.md § 3): relevo de coberturas solo sobre Poaceae de cosecha de secano (maíz y sorgo); Relevo de frijol en el maíz doblado al llegar a la madurez, práctica tradicional de primera y postrera en Honduras; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      }
    },
    {
      "id": "beans",
      "name": "Frijol",
      "family": "Fabaceae",
      "role": "harvest",
      "stages": {
        "value": [
          15,
          20,
          30,
          15
        ],
        "source": "Ciclo de 70 a 85 días de variedades hondureñas; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "kc": {
        "value": [
          0.4,
          1.15,
          0.35
        ],
        "source": "FAO-56 Tabla 12",
        "confidence": "A"
      },
      "rootDepthM": {
        "value": 0.8,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.45,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1.15,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "kyByPhase": {
        "value": [
          0.2,
          1.1,
          0.75,
          0.2
        ],
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación; fases asignadas a etapas FAO-56 según docs/fuentes.md",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          8,
          5,
          15,
          27,
          32
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 31,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          5.5,
          7,
          8.2
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          200,
          1800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          40,
          35
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 0.8,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.7,
          0.4,
          0.2,
          0.3
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera",
        "apante"
      ],
      "relay": {
        "value": {
          "guest": "grain"
        },
        "source": "Relevo de frijol en el maíz doblado al llegar a la madurez, práctica tradicional de primera y postrera en Honduras; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      }
    },
    {
      "id": "sorghum",
      "name": "Sorgo (maicillo)",
      "family": "Poaceae",
      "role": "harvest",
      "stages": {
        "value": [
          20,
          35,
          40,
          30
        ],
        "source": "FAO-56 Tabla 11 (USA, Pakistán, Mediterráneo)",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.3,
          1.05,
          0.55
        ],
        "source": "FAO-56 Tabla 12; Kc med punto medio de 1,00-1,10",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 1.5,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.55,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 0.9,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "kyByPhase": {
        "value": [
          0.2,
          0.55,
          0.45,
          0.2
        ],
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación; fases asignadas a etapas FAO-56 según docs/fuentes.md",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          10,
          12,
          22,
          35,
          40
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 38,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.3,
          5.5,
          7.5,
          8.7
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1200
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "moderate",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          15
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 1.5,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.5,
          0.25,
          0.35
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ],
      "relay": {
        "value": {
          "hosts": [
            "cover"
          ]
        },
        "source": "Decisión de alcance del 2026-11-15 (docs/calibracion.md § 3): relevo de coberturas solo sobre Poaceae de cosecha de secano (maíz y sorgo)",
        "confidence": "C"
      }
    },
    {
      "id": "rice",
      "name": "Arroz",
      "family": "Poaceae",
      "role": "harvest",
      "stages": {
        "value": [
          30,
          30,
          60,
          30
        ],
        "source": "FAO-56 Tabla 11 (Trópico)",
        "confidence": "B"
      },
      "kc": {
        "value": [
          1.05,
          1.2,
          0.75
        ],
        "source": "FAO-56 Tabla 12; Kc fin punto medio de 0,60-0,90",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 0.75,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.2,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          10,
          10,
          20,
          32,
          38
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 35,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4,
          5,
          7,
          8.5
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1000
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "required",
        "source": "Cultivo de riego o vega en Honduras; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "tolerant",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          13
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 3.5,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.5,
          0.25,
          0.35
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "verano"
      ]
    },
    {
      "id": "peanut",
      "name": "Maní",
      "family": "Fabaceae",
      "role": "harvest",
      "stages": {
        "value": [
          25,
          35,
          45,
          25
        ],
        "source": "FAO-56 Tabla 11 (región tropical)",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.4,
          1.15,
          0.6
        ],
        "source": "FAO-56 Tabla 12; Kc ini típico de cultivo de campo",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 0.8,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.5,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 0.7,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "kyByPhase": {
        "value": [
          0.2,
          0.8,
          0.6,
          0.2
        ],
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación; fases asignadas a etapas FAO-56 según docs/fuentes.md",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          10,
          12,
          22,
          32,
          38
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 35,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          5.5,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1200
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          80,
          40
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 1.2,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.7,
          0.4,
          0.2,
          0.4
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ]
    },
    {
      "id": "cowpea",
      "name": "Caupí (frijol de vara)",
      "family": "Fabaceae",
      "role": "harvest",
      "stages": {
        "value": [
          20,
          30,
          30,
          20
        ],
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "kc": {
        "value": [
          0.4,
          1.05,
          0.5
        ],
        "source": "FAO-56 Tabla 12 (Green gram and cowpeas)",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 0.8,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.45,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          10,
          12,
          22,
          35,
          40
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 37,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.3,
          5.5,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1200
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          60,
          35
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 0.8,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.7,
          0.35,
          0.15,
          0.3
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ]
    },
    {
      "id": "soybean",
      "name": "Soya",
      "family": "Fabaceae",
      "role": "harvest",
      "stages": {
        "value": [
          15,
          15,
          40,
          15
        ],
        "source": "FAO-56 Tabla 11 (Trópico)",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.4,
          1.15,
          0.5
        ],
        "source": "FAO-56 Tabla 12; Kc ini típico de cultivo de campo",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 1,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.5,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 0.85,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          10,
          10,
          20,
          30,
          38
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 35,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          6,
          7,
          8.5
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1200
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "moderate",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          100,
          60
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 1.5,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.7,
          0.4,
          0.2,
          0.4
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera"
      ]
    },
    {
      "id": "sesame",
      "name": "Ajonjolí",
      "family": "Pedaliaceae",
      "role": "harvest",
      "stages": {
        "value": [
          20,
          30,
          40,
          20
        ],
        "source": "FAO-56 Tabla 11 (China)",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.35,
          1.1,
          0.25
        ],
        "source": "FAO-56 Tabla 12; Kc ini típico de cultivo de campo",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 1.2,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.6,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          12,
          15,
          25,
          32,
          40
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 38,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          5.5,
          7.5,
          8.5
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          30
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 0.6,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.5,
          0.3,
          0.4
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "postrera"
      ]
    },
    {
      "id": "cassava",
      "name": "Yuca",
      "family": "Euphorbiaceae",
      "role": "harvest",
      "stages": {
        "value": [
          20,
          40,
          90,
          60
        ],
        "source": "FAO-56 Tabla 11 (año 1)",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.3,
          0.8,
          0.3
        ],
        "source": "FAO-56 Tabla 12 (año 1)",
        "confidence": "A"
      },
      "rootDepthM": {
        "value": 0.8,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.35,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          12,
          12,
          20,
          30,
          40
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 40,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4,
          5,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1500
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          4
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 10,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.5,
          0.3,
          0.4
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera"
      ]
    },
    {
      "id": "sweet_potato",
      "name": "Camote",
      "family": "Convolvulaceae",
      "role": "harvest",
      "stages": {
        "value": [
          15,
          30,
          50,
          30
        ],
        "source": "FAO-56 Tabla 11 (Trópico)",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.5,
          1.15,
          0.65
        ],
        "source": "FAO-56 Tabla 12; Kc ini típico",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 1.2,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.65,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          10,
          10,
          20,
          30,
          38
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 35,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          5.5,
          6.8,
          7.8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          4
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 8,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.7,
          0.3,
          0.15,
          0.2
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ]
    },
    {
      "id": "potato",
      "name": "Papa",
      "family": "Solanaceae",
      "role": "harvest",
      "stages": {
        "value": [
          25,
          30,
          30,
          20
        ],
        "source": "FAO-56 Tabla 11 (semiárido)",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.5,
          1.15,
          0.75
        ],
        "source": "FAO-56 Tabla 12; Kc ini típico",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 0.5,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.35,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1.1,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          4,
          5,
          14,
          20,
          28
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 27,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          5,
          6.5,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          1300,
          2800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          4
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 15,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.5,
          0.3,
          0.4
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ]
    },
    {
      "id": "tomato",
      "name": "Tomate",
      "family": "Solanaceae",
      "role": "harvest",
      "stages": {
        "value": [
          30,
          40,
          40,
          25
        ],
        "source": "FAO-56 Tabla 11",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.6,
          1.15,
          0.8
        ],
        "source": "FAO-56 Tabla 12; Kc fin punto medio de 0,70-0,90",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 1,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.4,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1.05,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          10,
          8,
          18,
          28,
          35
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 32,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          6,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          3
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 25,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.5,
          0.35,
          0.45
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "verano",
        "postrera"
      ]
    },
    {
      "id": "sweet_pepper",
      "name": "Chile dulce",
      "family": "Solanaceae",
      "role": "harvest",
      "stages": {
        "value": [
          25,
          35,
          40,
          20
        ],
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "kc": {
        "value": [
          0.6,
          1.05,
          0.9
        ],
        "source": "FAO-56 Tabla 12; Kc ini típico",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 0.8,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.3,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1.1,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          10,
          10,
          18,
          30,
          35
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 33,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          6,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          3
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 12,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.5,
          0.35,
          0.45
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "verano",
        "postrera"
      ]
    },
    {
      "id": "onion",
      "name": "Cebolla",
      "family": "Amaryllidaceae",
      "role": "harvest",
      "stages": {
        "value": [
          15,
          25,
          70,
          40
        ],
        "source": "FAO-56 Tabla 11",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.7,
          1.05,
          0.75
        ],
        "source": "FAO-56 Tabla 12; Kc ini típico",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 0.45,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.3,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1.1,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          7,
          5,
          13,
          27,
          35
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 32,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          6,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          2000
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          3
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 20,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.9,
          0.7,
          0.5,
          0.6
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "verano"
      ]
    },
    {
      "id": "cabbage",
      "name": "Repollo",
      "family": "Brassicaceae",
      "role": "harvest",
      "stages": {
        "value": [
          40,
          60,
          50,
          15
        ],
        "source": "FAO-56 Tabla 11",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.7,
          1.05,
          0.95
        ],
        "source": "FAO-56 Tabla 12; Kc ini típico",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 0.7,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.45,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 0.95,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          5,
          5,
          13,
          24,
          30
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 28,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          5,
          6,
          7.5,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          600,
          2500
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "moderate",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          3
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 25,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.4,
          0.2,
          0.3
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ]
    },
    {
      "id": "cucumber",
      "name": "Pepino",
      "family": "Cucurbitaceae",
      "role": "harvest",
      "stages": {
        "value": [
          20,
          30,
          40,
          15
        ],
        "source": "FAO-56 Tabla 11",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.6,
          1,
          0.75
        ],
        "source": "FAO-56 Tabla 12",
        "confidence": "A"
      },
      "rootDepthM": {
        "value": 1,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.5,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          12,
          12,
          20,
          30,
          35
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 33,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          5.5,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1500
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          3
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 15,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.45,
          0.3,
          0.4
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "verano",
        "postrera"
      ]
    },
    {
      "id": "watermelon",
      "name": "Sandía",
      "family": "Cucurbitaceae",
      "role": "harvest",
      "stages": {
        "value": [
          20,
          30,
          30,
          30
        ],
        "source": "FAO-56 Tabla 11",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.4,
          1,
          0.75
        ],
        "source": "FAO-56 Tabla 12",
        "confidence": "A"
      },
      "rootDepthM": {
        "value": 1.2,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.4,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1.1,
        "source": "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación",
        "confidence": "B"
      },
      "temperatureC": {
        "value": [
          12,
          12,
          22,
          32,
          38
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 36,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          5,
          6,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "required",
        "source": "Cultivo de riego o vega en Honduras; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          3
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 20,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.45,
          0.25,
          0.35
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "verano"
      ]
    },
    {
      "id": "melon",
      "name": "Melón",
      "family": "Cucurbitaceae",
      "role": "harvest",
      "stages": {
        "value": [
          30,
          45,
          35,
          10
        ],
        "source": "FAO-56 Tabla 11 (cantaloupe)",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.5,
          0.85,
          0.6
        ],
        "source": "FAO-56 Tabla 12 (cantaloupe)",
        "confidence": "A"
      },
      "rootDepthM": {
        "value": 1.2,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.4,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          12,
          12,
          22,
          32,
          38
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 36,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          5,
          6,
          7.5,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "required",
        "source": "Cultivo de riego o vega en Honduras; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          3
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 18,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.8,
          0.45,
          0.25,
          0.35
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "verano"
      ]
    },
    {
      "id": "squash",
      "name": "Ayote",
      "family": "Cucurbitaceae",
      "role": "harvest",
      "stages": {
        "value": [
          20,
          30,
          30,
          15
        ],
        "source": "FAO-56 Tabla 11",
        "confidence": "B"
      },
      "kc": {
        "value": [
          0.5,
          1,
          0.8
        ],
        "source": "FAO-56 Tabla 12 (Pumpkin, Winter Squash); Kc ini típico",
        "confidence": "B"
      },
      "rootDepthM": {
        "value": 1.2,
        "source": "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)",
        "confidence": "B"
      },
      "depletion": {
        "value": 0.35,
        "source": "FAO-56 Tabla 22",
        "confidence": "A"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          10,
          8,
          18,
          27,
          35
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 33,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          5.5,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "moderate",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          0,
          3
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 12,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.7,
          0.35,
          0.2,
          0.3
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ]
    },
    {
      "id": "mucuna",
      "name": "Mucuna (frijol terciopelo)",
      "family": "Fabaceae",
      "role": "cover",
      "stages": {
        "value": [
          20,
          40,
          60,
          30
        ],
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "kc": {
        "value": [
          0.4,
          1.15,
          0.5
        ],
        "source": "FAO-56 Tabla 12, fila Beans dry and Pulses como aproximación para leguminosa de cobertura",
        "confidence": "C"
      },
      "rootDepthM": {
        "value": 1,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "depletion": {
        "value": 0.5,
        "source": "FAO-56 Tabla 12, fila Beans dry and Pulses como aproximación para leguminosa de cobertura",
        "confidence": "C"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          10,
          10,
          20,
          30,
          38
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 38,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4,
          5,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1500
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "moderate",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          150,
          0
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 0,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.6,
          0.2,
          0.05,
          0.05
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ],
      "relay": {
        "value": {
          "guest": "cover"
        },
        "source": "Sistema maíz-mucuna (abonera) de Honduras: Buckles, Triomphe y Sain (1998), Cover crops in hillside agriculture, IDRC/CIMMYT; sin verificar contra la publicación",
        "confidence": "B"
      }
    },
    {
      "id": "canavalia",
      "name": "Canavalia",
      "family": "Fabaceae",
      "role": "cover",
      "stages": {
        "value": [
          20,
          40,
          60,
          30
        ],
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "kc": {
        "value": [
          0.4,
          1.15,
          0.5
        ],
        "source": "FAO-56 Tabla 12, fila Beans dry and Pulses como aproximación para leguminosa de cobertura",
        "confidence": "C"
      },
      "rootDepthM": {
        "value": 1,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "depletion": {
        "value": 0.5,
        "source": "FAO-56 Tabla 12, fila Beans dry and Pulses como aproximación para leguminosa de cobertura",
        "confidence": "C"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          12,
          12,
          20,
          32,
          40
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 40,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.3,
          5,
          7,
          8
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1500
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "moderate",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          120,
          0
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 0,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.6,
          0.25,
          0.08,
          0.08
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera",
        "postrera"
      ],
      "relay": {
        "value": {
          "guest": "cover"
        },
        "source": "Cobertura en relevo como la mucuna; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      }
    },
    {
      "id": "pigeon_pea",
      "name": "Gandul (frijol de palo)",
      "family": "Fabaceae",
      "role": "cover",
      "stages": {
        "value": [
          30,
          60,
          90,
          40
        ],
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "kc": {
        "value": [
          0.4,
          1,
          0.5
        ],
        "source": "FAO-56 Tabla 12, fila Beans dry and Pulses como aproximación para leguminosa de cobertura",
        "confidence": "C"
      },
      "rootDepthM": {
        "value": 1.5,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "depletion": {
        "value": 0.5,
        "source": "FAO-56 Tabla 12, fila Beans dry and Pulses como aproximación para leguminosa de cobertura",
        "confidence": "C"
      },
      "kyTotal": {
        "value": 1,
        "source": "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "temperatureC": {
        "value": [
          10,
          10,
          20,
          35,
          40
        ],
        "source": "Temperaturas base, absolutas y óptimas: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "heatStressTMaxC": {
        "value": 40,
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "ph": {
        "value": [
          4.5,
          5,
          7,
          8.4
        ],
        "source": "Rangos tipo Ecocrop (opinión experta); Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "altitudeM": {
        "value": [
          0,
          1800
        ],
        "source": "Rango de altitud en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "irrigation": {
        "value": "optional",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "waterlogging": {
        "value": "sensitive",
        "source": "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "nitrogenKgHa": {
        "value": [
          90,
          0
        ],
        "source": "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "referenceYieldTHa": {
        "value": 0,
        "source": "Rendimiento de referencia en Honduras: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "coverFactor": {
        "value": [
          0.7,
          0.35,
          0.15,
          0.15
        ],
        "source": "Factor C de RUSLE por etapa: Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      },
      "seasonsHonduras": [
        "primera"
      ],
      "relay": {
        "value": {
          "guest": "cover"
        },
        "source": "Cobertura en relevo como la mucuna; Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)",
        "confidence": "C"
      }
    }
  ],
  "templates": [
    {
      "id": "maize-beans",
      "name": "Maíz y frijol (tradicional)",
      "crops": [
        "maize",
        "beans"
      ],
      "relay": [
        false,
        false
      ],
      "mulch": false
    },
    {
      "id": "maize-sorghum",
      "name": "Maíz y sorgo (maicillo)",
      "crops": [
        "maize",
        "sorghum"
      ],
      "relay": [
        false,
        false
      ],
      "mulch": false
    },
    {
      "id": "maize-mucuna",
      "name": "Maíz con mucuna en relevo",
      "crops": [
        "maize",
        "mucuna"
      ],
      "relay": [
        false,
        true
      ],
      "mulch": false
    },
    {
      "id": "quesungual",
      "name": "Sistema agroforestal Quesungual (maíz y frijol con mulch)",
      "crops": [
        "maize",
        "beans"
      ],
      "relay": [
        false,
        false
      ],
      "mulch": true
    }
  ]
}
`
