#!/bin/sh
# Container startup: load the QA dataset if this version is not there yet
# (the seeder checks and skips on its own), then start the API.
# Set SEED_ON_STARTUP=false to start without seeding (for example in production).
set -e

if [ "${SEED_ON_STARTUP:-true}" = "true" ]; then
  /app/seed
fi

exec /app/app
