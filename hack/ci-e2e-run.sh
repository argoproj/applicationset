#!/bin/bash

if [ "$ARGOCD_IN_CI" != "true" ] ; then
    echo "This script should only be run from GitHub actions."
    exit 1
fi

set -e

# Add ~/go/bin to PATH --------------------------------------------------------
# Add /usr/local/bin to PATH

export PATH=/home/runner/go/bin:/usr/local/bin:$PATH


# Run E2E test suite ----------------------------------------------------------
set -x
cd "$GITHUB_WORKSPACE/applicationset"
kubectl apply -f manifests/crds/argoproj.io_applicationsets.yaml
make build
make start-e2e 2>&1 | tee /tmp/appset-e2e-server.log &
# Uncomment this to see the Argo CD output alongside test output:
# tail -f /tmp/e2e-server.log &
make test-e2e
