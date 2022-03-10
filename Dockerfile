# https://github.com/argoproj/argo-cd/pull/8516 now requires us to copy Argo CD binary into the ApplicationSet controller


# Build the binary
FROM docker.io/library/golang:1.17.6 as builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod .
COPY go.sum .
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

RUN rm -f ./bin/*

# Build
RUN make build

FROM docker.io/library/ubuntu:21.10

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get dist-upgrade -y && \
  apt-get install -y git git-lfs gpg tini && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists /var/cache/apt/archives /tmp/* /var/tmp/*


# Add Argo CD helper scripts that are required by 'github.com/argoproj/argo-cd/util/git' package
COPY hack/from-argo-cd/gpg-wrapper.sh /usr/local/bin/gpg-wrapper.sh
COPY hack/from-argo-cd/git-verify-wrapper.sh /usr/local/bin/git-verify-wrapper.sh
COPY hack/from-argo-cd/git-ask-pass.sh /usr/local/bin/git-ask-pass.sh

COPY entrypoint.sh /usr/local/bin/entrypoint.sh

# Support for mounting configuration from a configmap
RUN mkdir -p /app/config/ssh && \
    touch /app/config/ssh/ssh_known_hosts && \
    ln -s /app/config/ssh/ssh_known_hosts /etc/ssh/ssh_known_hosts

RUN mkdir -p /app/config/tls
RUN mkdir -p /app/config/gpg/source && \
    mkdir -p /app/config/gpg/keys
#    chown argocd /app/config/gpg/keys && \
#    chmod 0700 /app/config/gpg/keys

WORKDIR /
COPY --from=builder /workspace/dist/argocd-applicationset /usr/local/bin/applicationset-controller
