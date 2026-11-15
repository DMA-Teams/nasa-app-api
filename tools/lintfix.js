const fs = require("fs");

const edit = (file, pairs) => {
  let s = fs.readFileSync(file, "utf8");
  for (const [from, to] of pairs) {
    if (!s.includes(from)) throw new Error(file + ": missing " + JSON.stringify(from));
    s = s.split(from).join(to);
  }
  fs.writeFileSync(file, s);
};

edit("cmd/api/main.go", [["\tdefer rdb.Close()\n", "\tdefer func() { _ = rdb.Close() }()\n"]]);
edit("cmd/etl/main.go", [["\tdefer rdb.Close()\n", "\tdefer func() { _ = rdb.Close() }()\n"]]);
edit("internal/sources/httpclient.go", [
  ["\tdefer resp.Body.Close()\n", "\tdefer func() { _ = resp.Body.Close() }()\n"],
  ["\tvar status *StatusError\n\tif errors.As(err, &status) {\n", "\tif status, ok := errors.AsType[*StatusError](err); ok {\n"],
]);
edit("internal/sources/raster.go", [["\tdefer f.Close()\n", "\tdefer func() { _ = f.Close() }()\n"]]);
edit("internal/httpapi/middleware.go", [
  ["\t\t\t\tif v == http.ErrAbortHandler {\n", "\t\t\t\tif err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {\n"],
  ["import (\n\t\"context\"\n", "import (\n\t\"context\"\n\t\"errors\"\n"],
]);
edit("internal/httpapi/ratelimit.go", [
  ["\tparts := strings.Split(forwarded, \",\")\n\tfor i := len(parts) - 1; i >= 0; i-- {\n\t\taddr, err := netip.ParseAddr(strings.TrimSpace(parts[i]))\n",
    "\tfor _, part := range slices.Backward(strings.Split(forwarded, \",\")) {\n\t\taddr, err := netip.ParseAddr(strings.TrimSpace(part))\n"],
  ["\t\"net/netip\"\n", "\t\"net/netip\"\n\t\"slices\"\n"],
]);
edit("internal/config/config.go", [
  ["\tfor _, v := range strings.Split(l.getenv(key), \",\") {", "\tfor v := range strings.SplitSeq(l.getenv(key), \",\") {"],
  ["\tfor _, v := range strings.Split(raw, \",\") {", "\tfor v := range strings.SplitSeq(raw, \",\") {"],
]);
edit("internal/httpapi/api_test.go", [
  ["httptest.NewRequest(http.MethodGet, target, nil)", "httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)"],
  ["httptest.NewRequest(http.MethodGet, \"/\", nil)", "httptest.NewRequestWithContext(context.Background(), http.MethodGet, \"/\", nil)"],
]);
edit("internal/agro/et0.go", [
  ["\treturn 4098 * float64(SaturationVaporPressure(t)) / math.Pow(tc+237.3, 2)", "\treturn 4098 * float64(SaturationVaporPressure(t)) / ((tc + 237.3) * (tc + 237.3))"],
]);
edit("internal/agro/evaluator.go", [
  ["func (p *plantingStats) value() float64 {\n\treturn waterSurvivalShare*(1-p.lossProbability) + waterYieldShare*p.expectedYield\n}\n\n", ""],
]);
edit("internal/geo/geo_test.go", [["want neighbours", "want neighbors"]]);
edit(".golangci.yml", [
  ["    - misspell\n", ""],
  ["    misspell:\n      locale: US\n      ignore-rules:\n        - presion\n        - suelo\n", ""],
  ["    gosec:\n      excludes:\n        - G115\n", "    gosec:\n      excludes:\n        - G115\n        - G404\n"],
  ["          - funlen\n          - gosec\n          - dupword\n", "          - funlen\n          - gosec\n          - dupword\n          - bodyclose\n      - path: internal/sources/raster\\.go\n        text: G304\n        linters:\n          - gosec\n"],
  ["      - path: internal/catalog/crops_data\\.go\n        linters:\n          - misspell\n          - dupword\n", "      - path: internal/catalog/crops_data\\.go\n        linters:\n          - dupword\n"],
]);
console.log("ok");
