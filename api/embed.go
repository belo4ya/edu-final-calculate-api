package api

import _ "embed"

//go:embed api.swagger.json
var OpenAPISpec []byte
