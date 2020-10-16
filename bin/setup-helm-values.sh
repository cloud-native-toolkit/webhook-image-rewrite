#!/usr/bin/env bash

REPOSITORY="$1"
IMAGE_TAG="$2"

export REPOSITORY IMAGE_TAG

yq w - 'webhook.image.repository' "${REPOSITORY}" | \
  yq w - 'webhook.image.tag' "${IMAGE_TAG}"
