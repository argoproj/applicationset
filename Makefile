VERSION?=$(shell cat VERSION)
IMAGE_NAMESPACE?=argoprojlabs
IMAGE_NAME?=argocd-applicationset
IMAGE_TAG?=latest
CONTAINER_REGISTRY?=

MKDOCS_DOCKER_IMAGE?=squidfunk/mkdocs-material:4.1.1
MKDOCS_RUN_ARGS?=

CURRENT_DIR=$(shell pwd)


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
	CGO_ENABLED=0 go build -ldflags="-w -s" -o ./dist/argocd-applicationset .

.PHONY: test
test: generate fmt vet manifests
	go test -race -count=1 -coverprofile=coverage.out `go list ./... | grep -v 'test/e2e'`

.PHONY: image
image: generate fmt vet
	docker build -t ${IMAGE_PREFIX}${IMAGE_NAME}:${IMAGE_TAG} .

.PHONY: image-push
image-push: image
	docker push ${IMAGE_PREFIX}${IMAGE_NAME}:${IMAGE_TAG}

.PHONY: deploy
deploy: manifests
	kustomize build manifests/namespace-install | kubectl apply -f -
	kubectl patch deployment -n argocd argocd-applicationset-controller --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "$(IMAGE)"}]'

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: generate
	$(CONTROLLER_GEN) crd:crdVersions=v1 paths="./..." output:crd:artifacts:config=./manifests/crds/
	hack/generate-manifests.sh

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
start-e2e:
	NAMESPACE=argocd-e2e "dist/argocd-applicationset" --metrics-addr=:12345 --probe-addr=:12346 --argocd-repo-server=localhost:8081 --namespace=argocd-e2e

# Begin the tests, targetting the standalone controller (started by make start-e2e) and the e2e argo-cd (started by make start-e2e)
.PHONY: test-e2e
test-e2e:
	NAMESPACE=argocd-e2e go test -race -count=1 -v -timeout 480s ./test/e2e/applicationset

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

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


