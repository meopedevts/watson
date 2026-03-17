#!/bin/sh
set -e

# Configure git identity for merge operations (conflict detection).
# Set GIT_USER_NAME and GIT_USER_EMAIL in your .env file.
if [ -n "$GIT_USER_NAME" ]; then
  git config --global user.name "$GIT_USER_NAME"
fi

if [ -n "$GIT_USER_EMAIL" ]; then
  git config --global user.email "$GIT_USER_EMAIL"
fi

exec watson "$@"
