#!/bin/bash

if [ "$ARGOCD_IN_CI" != "true" ] ; then
    echo "This script should only be run from GitHub actions."
    exit 1
fi

set -e

# GH actions workaround - Kill XSP4 process -----------------------------------


# Add same workaround for port 8084 as argoproj/argo-cd #5658
sudo pkill mono || true

# Install K3S -----------------------------------------------------------------
echo "Installing k3s"
set -x
curl -sfL https://get.k3s.io | sh -
sudo chmod -R a+rw /etc/rancher/k3s
sudo mkdir -p $HOME/.kube && sudo chown -R runner $HOME/.kube
sudo k3s kubectl config view --raw > $HOME/.kube/config
sudo chown runner $HOME/.kube/config
kubectl version
set +x

# Add ~/go/bin to PATH --------------------------------------------------------
# Add /usr/local/bin to PATH

export PATH=/home/runner/go/bin:/usr/local/bin:$PATH

# Download Go dependencies ----------------------------------------------------
go mod download
go get github.com/mattn/goreman

# Install all tools required for building & testing ---------------------------

cd "$GITHUB_WORKSPACE/argo-cd"

echo "Install all tools required for building & testing"
make install-test-tools-local


# Setup git username and email ------------------------------------------------
git config --global user.name "John Doe"
git config --global user.email "john.doe@example.com"

# Pull Docker image required for tests ----------------------------------------
docker pull quay.io/dexidp/dex:v2.25.0
docker pull argoproj/argo-cd-ci-builder:v1.0.0
docker pull redis:6.2.4-alpine
      
# Create target directory for binaries in the build-process -------------------
mkdir -p dist
chown runner dist


# Run Argo CD E2E server and wait for it being available ----------------------

echo "Run Argo CD E2E server and wait for it being available"

set -x
# Something is weird in GH runners -- there's a phantom listener for
# port 8080 which is not visible in netstat -tulpen, but still there
# with a HTTP listener. We have API server listening on port 8088
# instead.
make start-e2e-local 2>&1 | sed -r "s/[[:cntrl:]]\[[0-9]{1,3}m//g" > /tmp/e2e-server.log &
count=1
until curl -f http://127.0.0.1:8088/healthz; do
    sleep 10;
    if test $count -ge 180; then
        echo "Timeout"
        exit 1
    fi
    count=$((count+1))
done
