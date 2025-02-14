#!/bin/sh

set -e

if [  "$1" = "api" ]; then
  api

else
  exec "$@"
fi;
