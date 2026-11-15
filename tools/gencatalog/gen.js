const T11 = "FAO-56 Tabla 11";
const T12 = "FAO-56 Tabla 12";
const T22 = "FAO-56 Tabla 22 (valor elegido dentro del rango para secano)";
const FAO33 = "FAO-33 (Doorenbos y Kassam, 1979); sin verificar contra la publicación";
const EST = "Estimación provisional del equipo, sin fuente; pendiente de validar (DICTA, Zamorano o literatura)";
const PROXY = "FAO-56 Tabla 12, fila Beans dry and Pulses como aproximación para leguminosa de cobertura";
const f = (value, source, confidence) => ({ value, source, confidence });

const rows = [
  ["maize", "Maíz", "Poaceae", "harvest", [18, 31, 36, 27], T11 + " (Nigeria, húmedo, 125 días) con las mismas proporciones, acortado a 112 días para que a ~1.000 m (San Andrés, Lempira) la primera sembrada a mediados de mayo se coseche a fines de septiembre, como en los calendarios de primera (siembra mayo-junio, cosecha septiembre); ajuste provisional, a calibrar con NDVI (docs/calibracion.md, etapa 2)", "C", [0.3, 1.2, 0.35], T12 + "; Kc ini típico de cultivo de campo", "B", 1.2, 0.55, 1.25, "B", [0.4, 1.5, 0.5, 0.2], [10, 10, 18, 32, 40], 35, [4.5, 5.5, 7.5, 8.5], [0, 2200], "optional", "moderate", [0, 16], 1.8, [0.8, 0.5, 0.25, 0.35], ["primera", "postrera"]],
  ["beans", "Frijol", "Fabaceae", "harvest", [15, 20, 30, 15], "Ciclo de 70 a 85 días de variedades hondureñas; " + EST, "C", [0.4, 1.15, 0.35], T12, "A", 0.8, 0.45, 1.15, "B", [0.2, 1.1, 0.75, 0.2], [8, 5, 15, 27, 32], 31, [4.5, 5.5, 7.0, 8.2], [200, 1800], "optional", "sensitive", [40, 35], 0.8, [0.7, 0.4, 0.2, 0.3], ["primera", "postrera", "apante"]],
  ["sorghum", "Sorgo (maicillo)", "Poaceae", "harvest", [20, 35, 40, 30], T11 + " (USA, Pakistán, Mediterráneo)", "B", [0.3, 1.05, 0.55], T12 + "; Kc med punto medio de 1,00-1,10", "B", 1.5, 0.55, 0.9, "B", [0.2, 0.55, 0.45, 0.2], [10, 12, 22, 35, 40], 38, [4.3, 5.5, 7.5, 8.7], [0, 1200], "optional", "moderate", [0, 15], 1.5, [0.8, 0.5, 0.25, 0.35], ["primera", "postrera"]],
  ["rice", "Arroz", "Poaceae", "harvest", [30, 30, 60, 30], T11 + " (Trópico)", "B", [1.05, 1.2, 0.75], T12 + "; Kc fin punto medio de 0,60-0,90", "B", 0.75, 0.2, 1.0, "C", null, [10, 10, 20, 32, 38], 35, [4.0, 5.0, 7.0, 8.5], [0, 1000], "required", "tolerant", [0, 13], 3.5, [0.8, 0.5, 0.25, 0.35], ["primera", "verano"]],
  ["peanut", "Maní", "Fabaceae", "harvest", [25, 35, 45, 25], T11 + " (región tropical)", "B", [0.4, 1.15, 0.6], T12 + "; Kc ini típico de cultivo de campo", "B", 0.8, 0.5, 0.7, "B", [0.2, 0.8, 0.6, 0.2], [10, 12, 22, 32, 38], 35, [4.5, 5.5, 7.0, 8.0], [0, 1200], "optional", "sensitive", [80, 40], 1.2, [0.7, 0.4, 0.2, 0.4], ["primera", "postrera"]],
  ["cowpea", "Caupí (frijol de vara)", "Fabaceae", "harvest", [20, 30, 30, 20], EST, "C", [0.4, 1.05, 0.5], T12 + " (Green gram and cowpeas)", "B", 0.8, 0.45, 1.0, "C", null, [10, 12, 22, 35, 40], 37, [4.3, 5.5, 7.0, 8.0], [0, 1200], "optional", "sensitive", [60, 35], 0.8, [0.7, 0.35, 0.15, 0.3], ["primera", "postrera"]],
  ["soybean", "Soya", "Fabaceae", "harvest", [15, 15, 40, 15], T11 + " (Trópico)", "B", [0.4, 1.15, 0.5], T12 + "; Kc ini típico de cultivo de campo", "B", 1.0, 0.5, 0.85, "B", null, [10, 10, 20, 30, 38], 35, [4.5, 6.0, 7.0, 8.5], [0, 1200], "optional", "moderate", [100, 60], 1.5, [0.7, 0.4, 0.2, 0.4], ["primera"]],
  ["sesame", "Ajonjolí", "Pedaliaceae", "harvest", [20, 30, 40, 20], T11 + " (China)", "B", [0.35, 1.1, 0.25], T12 + "; Kc ini típico de cultivo de campo", "B", 1.2, 0.6, 1.0, "C", null, [12, 15, 25, 32, 40], 38, [4.5, 5.5, 7.5, 8.5], [0, 800], "optional", "sensitive", [0, 30], 0.6, [0.8, 0.5, 0.3, 0.4], ["postrera"]],
  ["cassava", "Yuca", "Euphorbiaceae", "harvest", [20, 40, 90, 60], T11 + " (año 1)", "B", [0.3, 0.8, 0.3], T12 + " (año 1)", "A", 0.8, 0.35, 1.0, "C", null, [12, 12, 20, 30, 40], 40, [4.0, 5.0, 7.0, 8.0], [0, 1500], "optional", "sensitive", [0, 4], 10, [0.8, 0.5, 0.3, 0.4], ["primera"]],
  ["sweet_potato", "Camote", "Convolvulaceae", "harvest", [15, 30, 50, 30], T11 + " (Trópico)", "B", [0.5, 1.15, 0.65], T12 + "; Kc ini típico", "B", 1.2, 0.65, 1.0, "C", null, [10, 10, 20, 30, 38], 35, [4.5, 5.5, 6.8, 7.8], [0, 1800], "optional", "sensitive", [0, 4], 8, [0.7, 0.3, 0.15, 0.2], ["primera", "postrera"]],
  ["potato", "Papa", "Solanaceae", "harvest", [25, 30, 30, 20], T11 + " (semiárido)", "B", [0.5, 1.15, 0.75], T12 + "; Kc ini típico", "B", 0.5, 0.35, 1.1, "B", null, [4, 5, 14, 20, 28], 27, [4.5, 5.0, 6.5, 8.0], [1300, 2800], "optional", "sensitive", [0, 4], 15, [0.8, 0.5, 0.3, 0.4], ["primera", "postrera"]],
  ["tomato", "Tomate", "Solanaceae", "harvest", [30, 40, 40, 25], T11, "B", [0.6, 1.15, 0.8], T12 + "; Kc fin punto medio de 0,70-0,90", "B", 1.0, 0.4, 1.05, "B", null, [10, 8, 18, 28, 35], 32, [4.5, 6.0, 7.0, 8.0], [0, 1800], "optional", "sensitive", [0, 3], 25, [0.8, 0.5, 0.35, 0.45], ["verano", "postrera"]],
  ["sweet_pepper", "Chile dulce", "Solanaceae", "harvest", [25, 35, 40, 20], EST, "C", [0.6, 1.05, 0.9], T12 + "; Kc ini típico", "B", 0.8, 0.3, 1.1, "B", null, [10, 10, 18, 30, 35], 33, [4.5, 6.0, 7.0, 8.0], [0, 1800], "optional", "sensitive", [0, 3], 12, [0.8, 0.5, 0.35, 0.45], ["verano", "postrera"]],
  ["onion", "Cebolla", "Amaryllidaceae", "harvest", [15, 25, 70, 40], T11, "B", [0.7, 1.05, 0.75], T12 + "; Kc ini típico", "B", 0.45, 0.3, 1.1, "B", null, [7, 5, 13, 27, 35], 32, [4.5, 6.0, 7.0, 8.0], [0, 2000], "optional", "sensitive", [0, 3], 20, [0.9, 0.7, 0.5, 0.6], ["verano"]],
  ["cabbage", "Repollo", "Brassicaceae", "harvest", [40, 60, 50, 15], T11, "B", [0.7, 1.05, 0.95], T12 + "; Kc ini típico", "B", 0.7, 0.45, 0.95, "B", null, [5, 5, 13, 24, 30], 28, [5.0, 6.0, 7.5, 8.0], [600, 2500], "optional", "moderate", [0, 3], 25, [0.8, 0.4, 0.2, 0.3], ["primera", "postrera"]],
  ["cucumber", "Pepino", "Cucurbitaceae", "harvest", [20, 30, 40, 15], T11, "B", [0.6, 1.0, 0.75], T12, "A", 1.0, 0.5, 1.0, "C", null, [12, 12, 20, 30, 35], 33, [4.5, 5.5, 7.0, 8.0], [0, 1500], "optional", "sensitive", [0, 3], 15, [0.8, 0.45, 0.3, 0.4], ["verano", "postrera"]],
  ["watermelon", "Sandía", "Cucurbitaceae", "harvest", [20, 30, 30, 30], T11, "B", [0.4, 1.0, 0.75], T12, "A", 1.2, 0.4, 1.1, "B", null, [12, 12, 22, 32, 38], 36, [5.0, 6.0, 7.0, 8.0], [0, 800], "required", "sensitive", [0, 3], 20, [0.8, 0.45, 0.25, 0.35], ["verano"]],
  ["melon", "Melón", "Cucurbitaceae", "harvest", [30, 45, 35, 10], T11 + " (cantaloupe)", "B", [0.5, 0.85, 0.6], T12 + " (cantaloupe)", "A", 1.2, 0.4, 1.0, "C", null, [12, 12, 22, 32, 38], 36, [5.0, 6.0, 7.5, 8.0], [0, 800], "required", "sensitive", [0, 3], 18, [0.8, 0.45, 0.25, 0.35], ["verano"]],
  ["squash", "Ayote", "Cucurbitaceae", "harvest", [20, 30, 30, 15], T11, "B", [0.5, 1.0, 0.8], T12 + " (Pumpkin, Winter Squash); Kc ini típico", "B", 1.2, 0.35, 1.0, "C", null, [10, 8, 18, 27, 35], 33, [4.5, 5.5, 7.0, 8.0], [0, 1800], "optional", "moderate", [0, 3], 12, [0.7, 0.35, 0.2, 0.3], ["primera", "postrera"]],
  ["mucuna", "Mucuna (frijol terciopelo)", "Fabaceae", "cover", [20, 40, 60, 30], EST, "C", [0.4, 1.15, 0.5], PROXY, "C", 1.0, 0.5, 1.0, "C", null, [10, 10, 20, 30, 38], 38, [4.0, 5.0, 7.0, 8.0], [0, 1500], "optional", "moderate", [150, 0], 0, [0.6, 0.2, 0.05, 0.05], ["primera", "postrera"]],
  ["canavalia", "Canavalia", "Fabaceae", "cover", [20, 40, 60, 30], EST, "C", [0.4, 1.15, 0.5], PROXY, "C", 1.0, 0.5, 1.0, "C", null, [12, 12, 20, 32, 40], 40, [4.3, 5.0, 7.0, 8.0], [0, 1500], "optional", "moderate", [120, 0], 0, [0.6, 0.25, 0.08, 0.08], ["primera", "postrera"]],
  ["pigeon_pea", "Gandul (frijol de palo)", "Fabaceae", "cover", [30, 60, 90, 40], EST, "C", [0.4, 1.0, 0.5], PROXY, "C", 1.5, 0.5, 1.0, "C", null, [10, 10, 20, 35, 40], 40, [4.5, 5.0, 7.0, 8.4], [0, 1800], "optional", "sensitive", [90, 0], 0, [0.7, 0.35, 0.15, 0.15], ["primera"]],
];

const RELAY_SCOPE = "Decisión de alcance del 2026-11-15 (docs/calibracion.md § 3): relevo de coberturas solo sobre Poaceae de cosecha de secano (maíz y sorgo)";
const RELAY_GRAIN = "Relevo de frijol en el maíz doblado al llegar a la madurez, práctica tradicional de primera y postrera en Honduras; " + EST;
const MUCUNA_MAIZE = "Sistema maíz-mucuna (abonera) de Honduras: Buckles, Triomphe y Sain (1998), Cover crops in hillside agriculture, IDRC/CIMMYT; sin verificar contra la publicación";
const relay = {
  maize: f({ hosts: ["cover", "grain"] }, RELAY_SCOPE + "; " + RELAY_GRAIN, "C"),
  sorghum: f({ hosts: ["cover"] }, RELAY_SCOPE, "C"),
  beans: f({ guest: "grain" }, RELAY_GRAIN, "C"),
  mucuna: f({ guest: "cover" }, MUCUNA_MAIZE, "B"),
  canavalia: f({ guest: "cover" }, "Cobertura en relevo como la mucuna; " + EST, "C"),
  pigeon_pea: f({ guest: "cover" }, "Cobertura en relevo como la mucuna; " + EST, "C"),
};

const crops = rows.map((r) => {
  const [id, name, family, role, st, stSrc, stC, kc, kcSrc, kcC, root, dep, ky, kyC, kyP, temp, heat, ph, alt, irr, wl, n, yld, cover, seasons] = r;
  const proxyRoot = kcC === "C";
  const o = {
    id, name, family, role,
    stages: f(st, stSrc, stC),
    kc: f(kc, kcSrc, kcC),
    rootDepthM: f(root, proxyRoot ? EST : T22, proxyRoot ? "C" : "B"),
    depletion: f(dep, proxyRoot ? PROXY : "FAO-56 Tabla 22", proxyRoot ? "C" : "A"),
    kyTotal: f(ky, kyC === "C" ? "Sin Ky en FAO-33 para este cultivo; se asume 1,0 (respuesta proporcional). " + EST : FAO33, kyC),
  };
  if (kyP) o.kyByPhase = f(kyP, FAO33 + "; fases asignadas a etapas FAO-56 según docs/fuentes.md", "B");
  Object.assign(o, {
    temperatureC: f(temp, "Temperaturas base, absolutas y óptimas: " + EST, "C"),
    heatStressTMaxC: f(heat, EST, "C"),
    ph: f(ph, "Rangos tipo Ecocrop (opinión experta); " + EST, "C"),
    altitudeM: f(alt, "Rango de altitud en Honduras: " + EST, "C"),
    irrigation: f(irr, irr === "required" ? "Cultivo de riego o vega en Honduras; " + EST : EST, "C"),
    waterlogging: f(wl, EST, "C"),
    nitrogenKgHa: f(n, "Fijación y extracción de N: órdenes de magnitud del plan § 4.8; " + EST, "C"),
    referenceYieldTHa: f(yld, "Rendimiento de referencia en Honduras: " + EST, "C"),
    coverFactor: f(cover, "Factor C de RUSLE por etapa: " + EST, "C"),
    seasonsHonduras: seasons,
  });
  if (relay[id]) o.relay = relay[id];
  return o;
});

const templates = [
  { id: "maize-beans", name: "Maíz y frijol (tradicional)", crops: ["maize", "beans"], relay: [false, false], mulch: false },
  { id: "maize-sorghum", name: "Maíz y sorgo (maicillo)", crops: ["maize", "sorghum"], relay: [false, false], mulch: false },
  { id: "maize-mucuna", name: "Maíz con mucuna en relevo", crops: ["maize", "mucuna"], relay: [false, true], mulch: false },
  { id: "quesungual", name: "Sistema agroforestal Quesungual (maíz y frijol con mulch)", crops: ["maize", "beans"], relay: [false, false], mulch: true },
];

const json = JSON.stringify({ version: "2026-11-15", crops, templates }, null, 2);
if (json.includes("`")) throw new Error("backtick in catalog");
process.stdout.write("package catalog\n\nconst cropsJSON = `" + json + "\n`\n");
