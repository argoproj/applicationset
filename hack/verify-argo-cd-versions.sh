#!/bin/bash

# To adopt a new version of Argo CD:
# 1) Update this value to the GitHub tag of the target 'argoproj/argo-cd' release (example: 'v1.8.4'). 
# 2) Fix the errors that are reported below (by editing the version string in the file reported in the error)
TARGET_ARGO_CD_VERSION=v2.0.0

# Extract the Argo CD repository string from ci-build.yaml, which SHOULD contain the target Argo CD version
VERSION_FROM_CI_BUILD=$( awk '/BEGIN-ARGO-CD-VERSION/,/END-ARGO-CD-VERSION/' .github/workflows/ci-build.yaml )

if [[ $VERSION_FROM_CI_BUILD != *"$TARGET_ARGO_CD_VERSION"* ]]; then
    echo
    echo "ERROR: '.github/workflows/ci-build.yaml' does not target the expected Argo CD version: $TARGET_ARGO_CD_VERSION"
    echo "- Found: $VERSION_FROM_CI_BUILD"
    exit 1
fi

# Extract the argoproj/argo-cd GitHub resource URL, which SHOULD contain the target version
VERSION_FROM_KUSTOMIZATION_YAML=$( cat manifests/namespace-install-with-argo-cd/kustomization.yaml | grep "argoproj/argo-cd" )

if [[ $VERSION_FROM_KUSTOMIZATION_YAML != *"$TARGET_ARGO_CD_VERSION"* ]]; then
    echo
    echo "ERROR: 'manifests/namespace-install-with-argo-cd/kustomization.yaml' does not target the expected Argo CD version: $TARGET_ARGO_CD_VERSION"
    echo "- Found: $VERSION_FROM_KUSTOMIZATION_YAML"
    exit 1
fi
