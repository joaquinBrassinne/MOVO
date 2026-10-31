// Package api embeds the OpenAPI contract of the HTTP API so the binary can serve it
// without reading the file system.
package api

import _ "embed"

// Spec is the OpenAPI 3.0 document of the HTTP API, in YAML.
//
//go:embed openapi.yaml
var Spec []byte
