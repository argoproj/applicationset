VERSION?=$(shell cat VERSION)
IMAGE?=argoprojlabs/argocd-applicationset:v$(VERSION)
DOCKER_PUSH?=true

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o ./dist/argocd-applicationset .

.PHONY: image
image:
	docker build -t $(IMAGE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE) ; fi

.PHONY: deploy
deploy:
	kustomize build manifests/cluster-install | kubectl apply -f -
	kubectl patch deployment -n argocd argocd-applicationset-controller --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "$(IMAGE)"}]'

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
