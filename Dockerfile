# Build the binary
FROM golang:1.14.12 as builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod .
COPY go.sum .
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go .
COPY api/ api/
COPY pkg/ pkg/
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -a -o applicationset-controller main.go

FROM ubuntu:20.10

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get upgrade -y && \
  apt-get install -y git-all gpg && \
  rm -r /var/lib/apt/lists /var/cache/apt/archives

# Add Argo CD helper scripts that are required by 'github.com/argoproj/argo-cd/util/git' package
COPY hack/from-argo-cd/git-ask-pass.sh /usr/local/bin/git-ask-pass.sh
COPY hack/from-argo-cd/gpg-wrapper.sh /usr/local/bin/gpg-wrapper.sh
COPY hack/from-argo-cd/git-verify-wrapper.sh /usr/local/bin/git-verify-wrapper.sh


WORKDIR /
COPY --from=builder /workspace/applicationset-controller /usr/local/bin/
