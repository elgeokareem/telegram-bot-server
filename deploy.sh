#!/bin/bash

set -e

REPO_DIR="$HOME/documents/projects/botTelegram-server"
APP_DIR="/opt/botTelegram-server"
BINARY_NAME="bottelegram-server"
SERVICE_NAME="bottelegram-server.service"

echo "Changing to the repository directory..."
cd "$REPO_DIR" || exit 1

echo "Ensuring runtime PATH..."
export PATH="$PATH:/usr/local/go/bin"

echo "Pulling latest changes..."
git pull origin main

echo "Building the app..."
go mod tidy
go build -tags nomsgpack -o "$BINARY_NAME" .

echo "Stopping the service..."
if systemctl list-unit-files | grep -q "^${SERVICE_NAME}"; then
  systemctl stop "$SERVICE_NAME"
else
  pkill -f "$APP_DIR/$BINARY_NAME" || true
fi

echo "Copying the build"
mkdir -p "$APP_DIR"
cp "$REPO_DIR/$BINARY_NAME" "$APP_DIR/"
if [ -f "$REPO_DIR/.env" ]; then
  cp "$REPO_DIR/.env" "$APP_DIR/"
fi

echo "Restarting the app..."
chmod +x "$APP_DIR/$BINARY_NAME"
systemctl daemon-reload
if systemctl list-unit-files | grep -q "^${SERVICE_NAME}"; then
  systemctl restart "$SERVICE_NAME"
else
  mkdir -p "$APP_DIR/tmp"
  cd "$APP_DIR" || exit 1
  nohup "$APP_DIR/$BINARY_NAME" > "$APP_DIR/tmp/server.log" 2>&1 &
fi

echo "Deployment complete!"
