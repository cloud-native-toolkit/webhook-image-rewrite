# Image URL to use all building/pushing image targets;
# Use your own docker registry and image name for dev/test by overridding the
# IMAGE_REPO, IMAGE_NAME and IMAGE_TAG environment variable.
IMAGE_REPO ?= docker.io/ibmgaragecloud
IMAGE_NAME ?= image-rewrite
NAMESPACE ?= image-rewrite

# Github host to use for checking the source tree;
# Override this variable ue with your own value if you're working on forked repo.
GIT_HOST ?= github.com/ibm-garage-cloud

PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))

# Keep an existing GOPATH, make a private one if it is undefined
GOPATH_DEFAULT := $(PWD)/.go
export GOPATH ?= $(GOPATH_DEFAULT)
TESTARGS_DEFAULT := "-v"
export TESTARGS ?= $(TESTARGS_DEFAULT)
DEST := $(GOPATH)/src/$(GIT_HOST)/$(BASE_DIR)
IMAGE_TAG ?= $(shell date +v%Y%m%d)-$(shell git describe --match=$(git rev-parse --short=8 HEAD) --tags --always --dirty)


LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
    TARGET_OS ?= linux
    XARGS_FLAGS="-r"
else ifeq ($(LOCAL_OS),Darwin)
    TARGET_OS ?= darwin
    XARGS_FLAGS=
else
    $(error "This system's OS $(LOCAL_OS) isn't recognized/supported")
endif

all: fmt lint test build image setup

ifeq (,$(wildcard go.mod))
ifneq ("$(realpath $(DEST))", "$(realpath $(PWD))")
    $(error Please run 'make' from $(DEST). Current directory is $(PWD))
endif
endif

############################################################
# format section
############################################################

fmt:
	@echo "Run go fmt..."

############################################################
# lint section
############################################################

lint:
	@echo "Runing the golangci-lint..."

############################################################
# test section
############################################################

test:
	@echo "Running the tests for $(IMAGE_NAME)..."
	@go test $(TESTARGS) ./...

############################################################
# build section
############################################################

patch-ca-bundle-helm:
	cat ./chart/image-rewrite/values.yaml | \
		./bin/webhook-patch-ca-bundle.sh > \
		./chart/image-rewrite/values.yaml.tmp && \
		cp ./chart/image-rewrite/values.yaml.tmp ./chart/image-rewrite/values.yaml && \
		rm ./chart/image-rewrite/values.yaml.tmp

setup-image-helm:
	cat ./chart/image-rewrite/values.yaml | \
		./bin/setup-helm-values.sh $(IMAGE_REPO)/$(IMAGE_NAME) $(IMAGE_TAG) > \
		./chart/image-rewrite/values.yaml.tmp && \
		cp ./chart/image-rewrite/values.yaml.tmp ./chart/image-rewrite/values.yaml && \
		rm ./chart/image-rewrite/values.yaml.tmp

patch-ca-bundle-kustomize:
	cat ./kustomize/overlay/patches/mutatingwebhook.yaml | \
		./bin/webhook-patch-ca-bundle.sh > \
		./kustomize/overlay/patches/mutatingwebhook.yaml.tmp && \
		cp ./kustomize/overlay/patches/mutatingwebhook.yaml.tmp ./kustomize/overlay/patches/mutatingwebhook.yaml && \
		rm ./kustomize/overlay/patches/mutatingwebhook.yaml.tmp

setup-image-kustomize:
	cat ./kustomize/overlay/kustomization.yaml | \
		./bin/setup-kustomize.sh $(NAMESPACE) $(IMAGE_REPO)/$(IMAGE_NAME) $(IMAGE_TAG) > \
		./kustomize/overlay/kustomization.yaml.tmp && \
		cp ./kustomize/overlay/kustomization.yaml.tmp ./kustomize/overlay/kustomization.yaml && \
		rm ./kustomize/overlay/kustomization.yaml.tmp

setup-kustomize: patch-ca-bundle-kustomize setup-image-kustomize

setup-helm: patch-ca-bundle-helm setup-image-helm

setup: setup-kustomize

build:
	@echo "Building the $(IMAGE_NAME) binary..."
	@CGO_ENABLED=0 go build -o build/_output/bin/$(IMAGE_NAME) ./cmd/

############################################################
# image section
############################################################

image: build-image push-image

build-image: build
	@echo "Building the docker image: $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)..."
	@docker build -t $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) -f Dockerfile .

push-image: build-image
	@echo "Pushing the docker image for $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) and $(IMAGE_REPO)/$(IMAGE_NAME):latest..."
	@docker tag $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_REPO)/$(IMAGE_NAME):latest
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME):latest

deploy-helm:
	helm template image-rewrite chart/image-rewrite --namespace $(NAMESPACE) | kubectl apply -n $(NAMESPACE) -f -

deploy-kustomize:
	kustomize build kustomize/overlay | kubectl apply -f -

deploy: deploy-kustomize

############################################################
# clean section
############################################################
clean:
	@rm -rf build/_output

.PHONY: all fmt lint check test build image clean

