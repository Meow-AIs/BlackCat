// Package embed provides embedded assets for BlackCat.
// The ONNX embedding model and SQL schema are bundled into the binary
// via Go's embed directive, so no runtime downloads are needed.
package embed

import "embed"

// Schema contains the SQLite initialization SQL.
//
//go:embed schema/init.sql
var Schema string

// ModelFS contains the embedded ONNX model files.
// The model directory may be empty if building without the model
// (e.g., when using API-based embeddings or Ollama for embeddings).
//
//go:embed model
var ModelFS embed.FS

// ModelPath is the path within ModelFS to the ONNX model file.
const ModelPath = "model/minilm-l6-v2-int8.onnx"

// HasEmbeddedModel reports whether the ONNX model is bundled in this build.
func HasEmbeddedModel() bool {
	f, err := ModelFS.Open(ModelPath)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
