VERSION_PACKAGE=github.com/argoproj/applicationset/common
VERSION?=$(shell cat VERSION)
IMAGE_NAMESPACE?=argoproj
IMAGE_PLATFORMS?=linux/amd64,linux/arm64
IMAGE_NAME?=argocd-applicationset
IMAGE_TAG?=latest
CONTAINER_REGISTRY?=quay.io
GIT_COMMIT = $(shell git rev-parse HEAD)
LDFLAGS = -w -s -X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitCommit=${GIT_COMMIT}

MKDOCS_DOCKER_IMAGE?=squidfunk/mkdocs-material:4.1.1
MKDOCS_RUN_ARGS?=

CURRENT_DIR=$(shell pwd)

KUSTOMIZE = $(shell pwd)/bin/kustomize
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen

ifdef IMAGE_NAMESPACE

	ifdef CONTAINER_REGISTRY
		IMAGE_PREFIX=${CONTAINER_REGISTRY}/${IMAGE_NAMESPACE}/
	else
		IMAGE_PREFIX=${IMAGE_NAMESPACE}/
	endif

else
	IMAGE_PREFIX=
endif


# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: build
build: manifests fmt vet
	CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o ./dist/argocd-applicationset .

.PHONY: test
test: generate fmt vet manifests
	go test -race -count=1 -coverprofile=coverage.out `go list ./... | grep -v 'test/e2e'`

.PHONY: image
image: test
	docker buildx build --platform $(IMAGE_PLATFORMS) -t ${IMAGE_PREFIX}${IMAGE_NAME}:${IMAGE_TAG} .

.PHONY: image-push
image-push: image
	docker push ${IMAGE_PREFIX}${IMAGE_NAME}:${IMAGE_TAG}

.PHONY: deploy
deploy: kustomize manifests
	${KUSTOMIZE} build manifests/namespace-install | kubectl apply -f -
	kubectl patch deployment -n argocd argocd-applicationset-controller --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "$(IMAGE)"}]'

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: kustomize generate
	$(CONTROLLER_GEN) crd:crdVersions=v1,maxDescLen=0 paths="./..." output:crd:artifacts:config=./manifests/crds/
	KUSTOMIZE=${KUSTOMIZE} CONTAINER_REGISTRY=${CONTAINER_REGISTRY} hack/generate-manifests.sh

.PHONY: lint
lint:
	golangci-lint --version
	GOMAXPROCS=2 golangci-lint run --fix --verbose --timeout 300s

# Run go fmt against code
.PHONY: fmt
fmt:
	go fmt ./...

# Run go vet against code
.PHONY: vet
vet:
	go vet ./...

# Start the standalone controller for the purpose of running e2e tests
.PHONY: start-e2e
start-e2e: # Ensure the PlacementDecision CRD is present for the ClusterDecisionManagement tests
	kubectl apply -f https://raw.githubusercontent.com/open-cluster-management/api/a6845f2ebcb186ec26b832f60c988537a58f3859/cluster/v1alpha1/0000_04_clusters.open-cluster-management.io_placementdecisions.crd.yaml
	NAMESPACE=argocd-e2e "dist/argocd-applicationset" --metrics-addr=:12345 --probe-addr=:12346 --argocd-repo-server=localhost:8081 --namespace=argocd-e2e

# Begin the tests, targeting the standalone controller (started by make start-e2e) and the e2e argo-cd (started by make start-e2e)
.PHONY: test-e2e
test-e2e:
	NAMESPACE=argocd-e2e go test -race -count=1 -v -timeout 480s ./test/e2e/applicationset

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: build-docs-local
build-docs-local:
	mkdocs build

.PHONY: build-docs
build-docs:
	docker run ${MKDOCS_RUN_ARGS} --rm -it -p 8000:8000 -v ${CURRENT_DIR}:/docs ${MKDOCS_DOCKER_IMAGE} build

.PHONY: serve-docs-local
serve-docs-local:
	mkdocs serve

.PHONY: serve-docs
serve-docs:
	docker run ${MKDOCS_RUN_ARGS} --rm -it -p 8000:8000 -v ${CURRENT_DIR}:/docs ${MKDOCS_DOCKER_IMAGE} serve -a 0.0.0.0:8000

.PHONY: lint-docs
lint-docs:
	#  https://github.com/dkhamsing/awesome_bot
	find docs -name '*.md' -exec grep -l http {} + | xargs docker run --rm -v $(PWD):/mnt:ro dkhamsing/awesome_bot -t 3 --allow-dupe --allow-redirect --white-list `cat docs/assets/broken-link-ignore-list.txt | grep -v "#" | tr "\n" ','` --skip-save-results --


controller-gen: ## Download controller-gen to '(project root)/bin', if not already present.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0)


kustomize: ## Download kustomize to '(project root)/bin', if not already present.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.9.4)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
