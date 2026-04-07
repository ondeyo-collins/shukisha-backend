package swagger

import (
	"fmt"
	"net/http"
	"os"
)

// SwaggerHandler serves the Swagger UI HTML page
func SwaggerHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Shukisha API - Swagger UI</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/v1/openapi.yaml',
        dom_id: '#swagger-ui',
      });
    };
  </script>
</body>
</html>`
	fmt.Fprint(w, html)
}

// SpecHandler serves the openapi.yaml content
func SpecHandler(yamlPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, err := os.ReadFile(yamlPath)
		if err != nil {
			http.Error(w, "Specification not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/yaml")
		w.Write(content)
	}
}
