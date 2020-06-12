VERSION?=$(shell cat VERSION)
IMAGE_TAG?=v$(VERSION)
IMAGE_PREFIX?=argoprojlabs
DOCKER_PUSH?=true

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o ./dist/argocd-aplicationset .

.PHONY: image
image:
	docker build -t $(IMAGE_PREFIX)/argocd-aplicationset:$(IMAGE_TAG) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)/argocd-aplicationset:$(IMAGE_TAG) ; fi

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests:
	controller-gen paths=./api/... crd:trivialVersions=true output:dir=./manifests/crds/
	controller-gen object paths=./api/...

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...
