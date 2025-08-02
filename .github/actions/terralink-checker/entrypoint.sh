#!/bin/bash
set -e

DIRECTORY="$1"

# URL del release (ajusta la URL y el nombre del binario según corresponda)
RELEASE_URL="https://github.com/Isaac/terralink/releases/latest/download/terralink"

# Descargar el binario
curl -L "$RELEASE_URL" -o terralink
chmod +x terralink

# Ejecutar el comando
./terralink check --dir "$DIRECTORY"

