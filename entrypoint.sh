#!/bin/sh

set -e

if [  "$1" = "api" ]; then
  cd /app

  # Generate service discovery for custom Go services
  echo "Discovering services in /app/resources/data/services..."
  gen-discover /app/resources/data/services

  # Build the server with discovered services
  echo "Building server..."
  go build -mod=vendor -o /app/server ./cmd/server

  # Start the server (watcher is built-in, will auto-rebuild and restart)
  echo "Starting server with built-in hot-reload..."
  exec /app/server
else
  exec "$@"
fi;
