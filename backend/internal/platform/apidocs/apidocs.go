// Package apidocs serves the OpenAPI contract and a Swagger UI page to browse it.
//
// It is meant for development: the page loads Swagger UI from a CDN, and the process
// only mounts it when API_DOCS is enabled.
package apidocs

import (
	"net/http"
)

const (
	// PagePath is where the interactive page is served.
	PagePath = "/docs"
	// SpecPath is where the raw OpenAPI document is served.
	SpecPath = "/docs/openapi.yaml"

	// swaggerUIVersion is pinned so the page does not change under us.
	swaggerUIVersion = "5.17.14"
)

const page = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>MOVO API</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@` + swaggerUIVersion + `/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@` + swaggerUIVersion + `/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "` + SpecPath + `",
      dom_id: "#swagger-ui",
      persistAuthorization: true,
      tryItOutEnabled: true
    });
  </script>
</body>
</html>
`

// Handler serves the page at PagePath and spec at SpecPath; any other path is a 404.
func Handler(spec []byte) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+PagePath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	})
	mux.HandleFunc("GET "+SpecPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(spec)
	})
	return mux
}

// Mount returns a handler that serves the docs and hands every other request to next.
func Mount(spec []byte, next http.Handler) http.Handler {
	docs := Handler(spec)
	mux := http.NewServeMux()
	mux.Handle("GET "+PagePath, docs)
	mux.Handle("GET "+SpecPath, docs)
	mux.Handle("/", next)
	return mux
}
