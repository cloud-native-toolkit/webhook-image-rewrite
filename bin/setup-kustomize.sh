#!/usr/bin/env bash

NAMESPACE="$1"
IMAGE="$2"
IMAGE_TAG="$3"

export NAMESPACE IMAGE IMAGE_TAG

sed -E "s|(.*newName:).*|\1 ${IMAGE}|g" | sed -E "s|(.*newTag:).*|\1 ${IMAGE_TAG}|g" | sed -E "s|(.*namespace:).*|\1 ${NAMESPACE}|g"
