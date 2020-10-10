#!/usr/bin/env bash

NAMESPACE="$1"
IMAGE="$2"
IMAGE_TAG="$3"

export NAMESPACE IMAGE IMAGE_TAG

if command -v envsubst >/dev/null 2>&1; then
    envsubst
else
    sed -e "s|\${IMAGE}|${IMAGE}|g" | sed -e "s|\${IMAGE_TAG}|${IMAGE_TAG}|g" | sed -e "s|\${NAMESPACE}|${NAMESPACE}|g"
fi
