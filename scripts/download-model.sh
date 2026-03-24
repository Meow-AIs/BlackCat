#!/bin/bash
# Download the embedding model for bundling into the BlackCat binary.
# This script downloads the quantized MiniLM-L6-v2 ONNX model (~23MB)
# and places it in embed/model/ for go:embed.
#
# Usage: bash scripts/download-model.sh
#
# The model is only needed if you want local embeddings without Ollama.
# If you use Ollama or API-based embeddings, you can skip this.

set -euo pipefail

MODEL_DIR="embed/model"
MODEL_FILE="$MODEL_DIR/minilm-l6-v2-int8.onnx"
MODEL_URL="https://huggingface.co/Xenova/all-MiniLM-L6-v2/resolve/main/onnx/model_quantized.onnx"

if [ -f "$MODEL_FILE" ]; then
    echo "Model already exists: $MODEL_FILE ($(du -h "$MODEL_FILE" | cut -f1))"
    exit 0
fi

mkdir -p "$MODEL_DIR"

echo "Downloading MiniLM-L6-v2 (int8 quantized) ONNX model..."
echo "Source: $MODEL_URL"

if command -v curl &>/dev/null; then
    curl -fSL --progress-bar -o "$MODEL_FILE" "$MODEL_URL"
elif command -v wget &>/dev/null; then
    wget -q --show-progress -O "$MODEL_FILE" "$MODEL_URL"
else
    echo "Error: curl or wget required"
    exit 1
fi

SIZE=$(stat -f%z "$MODEL_FILE" 2>/dev/null || stat -c%s "$MODEL_FILE")
echo "Downloaded: $MODEL_FILE ($((SIZE / 1024 / 1024))MB)"
echo ""
echo "The model will be embedded in the binary on next build."
echo "Binary size will increase by ~$((SIZE / 1024 / 1024))MB."
