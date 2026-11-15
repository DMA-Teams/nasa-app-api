#!/usr/bin/env bash
set -euo pipefail

OUT="${1:-data/soil}"
BBOX="${BBOX:--89.40 12.95 -83.10 16.55}"
RES="0.0025 0.0025"
PROPERTIES="${PROPERTIES:-clay sand silt soc phh2o nitrogen cec bdod}"
DEPTHS="${DEPTHS:-0-5cm 5-15cm 15-30cm 30-60cm}"

mkdir -p "$OUT"

for prop in $PROPERTIES; do
  for depth in $DEPTHS; do
    name="${prop}_${depth}_mean"
    if [ -f "$OUT/$name.bin" ]; then
      echo "skip $name"
      continue
    fi

    echo "warp $name"
    docker run --rm -v "$(realpath "$OUT"):/out" ghcr.io/osgeo/gdal:alpine-small-3.11.4 \
      gdalwarp -overwrite -t_srs EPSG:4326 -te $BBOX -tr $RES -r average \
      -ot Int16 -dstnodata -32768 -of ENVI \
      "/vsicurl/https://files.isric.org/soilgrids/latest/data/${prop}/${name}.vrt" "/out/${name}.bin"
  done
done
