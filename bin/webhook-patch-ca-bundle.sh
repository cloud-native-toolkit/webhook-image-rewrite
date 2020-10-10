#!/bin/bash

NAMESPACE="$1"

set -o errexit
set -o nounset
set -o pipefail

CA_BUNDLE=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')

if [ -z "${CA_BUNDLE}" ]; then
    SECRET_NAMES=$(kubectl get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='default')].metadata.name}" | tr " " "\n")

    TMP_DIR="${PWD}/tmp-ca-bundle"
    mkdir -p "${TMP_DIR}"

    CA_BUNDLE_FILE="${TMP_DIR}/ca-bundle"

    echo "${SECRET_NAMES}" | while read secret; do
      ca_bundle=$(kubectl get secrets "${secret}" -o jsonpath="{.data.ca\.crt}" | base64 -D)
      if [[ -n "${ca_bundle}" ]]; then
        echo "${ca_bundle}" >> "${CA_BUNDLE_FILE}"
      fi
    done

    CA_BUNDLE=$(cat "${CA_BUNDLE_FILE}" | base64)

    rm -rf "${TMP_DIR}"
fi

export CA_BUNDLE NAMESPACE

if command -v envsubst >/dev/null 2>&1; then
    envsubst
else
    sed -e "s|\${CA_BUNDLE}|${CA_BUNDLE}|g" | sed -e "s|\${NAMESPACE}|${NAMESPACE}|g"
fi
