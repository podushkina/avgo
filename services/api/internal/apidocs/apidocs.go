package apidocs

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var spec []byte

func Spec() []byte { return spec }

const swaggerUIVersion = "5.30.2"

var page = []byte(`<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Безопасная сделка — API</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@` + swaggerUIVersion + `/swagger-ui.css">
<style>
  body { margin: 0; }
  .swagger-ui .topbar { display: none; }
  #offline {
    display: none;
    margin: 3rem auto;
    max-width: 40rem;
    padding: 1.5rem;
    border: 1px solid #e3e3e3;
    border-radius: 12px;
    font: 15px/1.55 -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, sans-serif;
  }
  #offline code { background: #f2f1f0; padding: 2px 6px; border-radius: 4px; }
</style>
</head>
<body>
<div id="swagger"></div>
<div id="offline">
  <h2>Swagger UI не загрузился</h2>
  <p>Интерфейс подтягивается с CDN, а до него сейчас нет доступа. Спецификация лежит рядом
     и доступна без интернета:</p>
  <p><a href="openapi.yaml">openapi.yaml</a> — открывается в
     <a href="https://editor.swagger.io">editor.swagger.io</a> или любом клиенте OpenAPI.</p>
  <p>Само API при этом работает: <code>curl localhost:8080/api/healthz</code></p>
</div>
<script src="https://unpkg.com/swagger-ui-dist@` + swaggerUIVersion + `/swagger-ui-bundle.js"
        onerror="document.getElementById('offline').style.display='block'"></script>
<script>
  window.addEventListener('load', function () {
    if (typeof SwaggerUIBundle !== 'function') {
      document.getElementById('offline').style.display = 'block';
      return;
    }
    SwaggerUIBundle({
      url: 'openapi.yaml',
      dom_id: '#swagger',
      docExpansion: 'list',
      defaultModelsExpandDepth: 1,
      tryItOutEnabled: true,
      withCredentials: true
    });
  });
</script>
</body>
</html>`)

func SpecHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(spec)
}

func UIHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
}
