package httpapi

import "net/http"

const (
	swaggerUIVersion  = "5.17.14"
	swaggerCSSSRI     = "sha384-wxLW6kwyHktdDGr6Pv1zgm/VGJh99lfUbzSn6HNHBENZlCN7W602k9VkGdxuFvPn"
	swaggerBundleSRI  = "sha384-wmyclcVGX/WhUkdkATwhaK1X1JtiNrr2EoYJ+diV3vj4v6OC5yCeSu+yW13SYJep"
	swaggerAssetsBase = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@" + swaggerUIVersion
)

const swaggerPage = `<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>nasa-app-api</title>
<link rel="stylesheet" href="` + swaggerAssetsBase + `/swagger-ui.css" integrity="` + swaggerCSSSRI + `" crossorigin="anonymous">
</head>
<body>
<div id="swagger-ui"></div>
<script src="` + swaggerAssetsBase + `/swagger-ui-bundle.js" integrity="` + swaggerBundleSRI + `" crossorigin="anonymous"></script>
<script>
window.onload = () => {
  window.ui = SwaggerUIBundle({ url: "/openapi.yaml", dom_id: "#swagger-ui", tryItOutEnabled: true });
};
</script>
</body>
</html>
`

func (a *API) openAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(openAPISpec))
}

func (a *API) swaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(swaggerPage))
}
