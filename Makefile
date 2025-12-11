# Makefile for infra-operator

# Image URL to use for building/pushing image targets
IMG ?= infra-operator:latest
REGISTRY ?= ttl.sh

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
CONTAINER_TOOL ?= docker

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/infra.operator main.go

.PHONY: run
run: fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: ## Build docker image.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image.
	$(CONTAINER_TOOL) push ${IMG}

.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for multiple platforms.
	$(CONTAINER_TOOL) buildx build --platform linux/amd64,linux/arm64 --push -t ${REGISTRY}/${IMG} .

##@ Deployment

.PHONY: install
install: ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f config/crd/bases/

.PHONY: deploy
deploy: ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/manager/namespace.yaml
	kubectl apply -f config/rbac/
	kubectl apply -f config/manager/deployment.yaml

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f config/manager/deployment.yaml
	kubectl delete -f config/rbac/
	kubectl delete -f config/manager/namespace.yaml

.PHONY: deploy-samples
deploy-samples: ## Deploy sample resources
	kubectl apply -f config/samples/

.PHONY: delete-samples
delete-samples: ## Delete sample resources
	kubectl delete -f config/samples/ --ignore-not-found=true

##@ Build Dependencies

.PHONY: mod-download
mod-download: ## Download Go modules
	go mod download

.PHONY: mod-tidy
mod-tidy: ## Tidy Go modules
	go mod tidy

.PHONY: mod-verify
mod-verify: ## Verify Go modules
	go mod verify

##@ Complete Installation

.PHONY: install-complete
install-complete: install deploy ## Install CRDs and deploy operator

.PHONY: uninstall-complete
uninstall-complete: undeploy uninstall ## Undeploy operator and uninstall CRDs
