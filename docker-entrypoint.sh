#!/bin/sh
set -e

# Configure git identity for merge operations (conflict detection).
# Set GIT_USER_NAME and GIT_USER_EMAIL in your .env file to override defaults.
GIT_NAME="${GIT_USER_NAME:-watson}"
GIT_EMAIL="${GIT_USER_EMAIL:-watson@localhost}"
git config --global user.name "$GIT_NAME"
git config --global user.email "$GIT_EMAIL"

exec watson "$@"
