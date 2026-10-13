#!/bin/sh
# Local production testing only: creates a self-signed certificate for
# localhost on first start (kept in the certs volume). For a real deployment,
# mount a real certificate at /etc/nginx/certs/tls.crt and tls.key instead.
set -eu
dir=/etc/nginx/certs
if [ ! -f "$dir/tls.crt" ] || [ ! -f "$dir/tls.key" ]; then
  mkdir -p "$dir"
  openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
    -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
    -keyout "$dir/tls.key" -out "$dir/tls.crt" 2>/dev/null
  chmod 600 "$dir/tls.key"
  echo "generated a self-signed certificate for localhost"
fi
