# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests:
	controller-gen paths=./api/... crd:trivialVersions=true output:dir=./manifests/crds/

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...
