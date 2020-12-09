VERSION?=$(shell cat VERSION)
IMAGE?=argoprojlabs/argocd-applicationset:v$(VERSION)
DOCKER_PUSH?=true

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o ./dist/argocd-applicationset .

.PHONY: test
test:
	go test -race -count=1 -coverprofile=coverage.out `go list ./...`

.PHONY: image
image:
	docker build -t $(IMAGE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE) ; fi

.PHONY: deploy
deploy:
	kustomize build manifests/namespace-install | kubectl apply -f -
	kubectl patch deployment -n argocd argocd-applicationset-controller --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "$(IMAGE)"}]'

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests:
	controller-gen paths=./api/... crd:trivialVersions=true output:dir=./manifests/crds/
	controller-gen object paths=./api/...

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
	NAMESPACE=argocd-e2e go test -race -count=1 -v -timeout 120s ./test/e2e/applicationset
