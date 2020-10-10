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

all: fmt lint test build image

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

create-signed-cert:
	./bin/webhook-create-signed-cert.sh \
    	--service $(IMAGE_NAME)-webhook-svc \
    	--secret $(IMAGE_NAME)-webhook-certs \
    	--namespace $(NAMESPACE)

patch-ca-bundle:
	cat ./deployment-template/mutatingwebhook.yaml | \
		./bin/webhook-patch-ca-bundle.sh $(NAMESPACE) > \
		./deployment/mutatingwebhook.yaml

setup-kustomize:
	cat ./deployment-template/kustomization.yaml | \
		./bin/setup-kustomize.sh $(NAMESPACE) $(IMAGE_REPO)/$(IMAGE_NAME) $(IMAGE_TAG) > ./deployment/kustomization.yaml

setup: create-signed-cert patch-ca-bundle setup-kustomize

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

deploy:
	kustomize build deployment/ | kubectl apply -f -

############################################################
# clean section
############################################################
clean:
	@rm -rf build/_output

.PHONY: all fmt lint check test build image clean

