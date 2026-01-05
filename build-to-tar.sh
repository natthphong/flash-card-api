#!/usr/bin/env bash
set -e

APP_NAME=flash-card
IMAGE_TAG=2
OUTPUT_TAR=${APP_NAME}-${IMAGE_TAG}.tar
IMAGE_NAME=${APP_NAME}:${IMAGE_TAG}
echo "==> Build Go binary"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o goapp

echo "==> Build docker image: ${IMAGE_NAME}"
docker build -t ${IMAGE_NAME} .

echo "==> Save docker image to tar: ${OUTPUT_TAR}"
docker save -o ${OUTPUT_TAR} ${IMAGE_NAME}

echo "==> Done"
echo "Image: ${IMAGE_NAME}"
echo "Tar  : ${OUTPUT_TAR}"


# Portainer
PORTAINER_BASE_URL="http://192.168.1.122:9002"
ENDPOINT_ID="3"
PORTAINER_API_KEY="${PORTAINER_API_KEY:-}"

echo "==> Upload tar to Portainer endpoint ${ENDPOINT_ID}"
curl -fS -X POST \
  "${PORTAINER_BASE_URL}/api/endpoints/${ENDPOINT_ID}/docker/v1.44/images/load" \
  -H "X-API-Key: ${PORTAINER_API_KEY}" \
  -H "Content-Type: application/x-tar" \
  --data-binary "@${OUTPUT_TAR}"

echo "==> Done"
echo "Image: ${IMAGE_NAME}"
echo "Tar  : ${OUTPUT_TAR}"