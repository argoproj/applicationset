
# Getting Started

This guide assumes you are familiar with Argo CD and its basic concepts. See the [Argo CD documentation](https://argoproj.github.io/argo-cd/core_concepts/) for more information.
    
## Requirements

* Installed [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`).

## Installation

### Install ApplicationSet into an existing Argo CD install

The ApplicationSet controller *must* be installed into the same namespace as the Argo CD it is targetting.

Presuming that Argo CD is installed into the `argocd` namespace, run the following command:

```bash
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/applicationset/master/manifests/install.yaml
```

Once installed, the ApplicationSet controller requires no additional setup.

The `manifests/install.yaml` file contains the Kubernetes manifests required to install the ApplicationSet controller:

- CustomResourceDefinition for `ApplicationSet` resource
- Deployment for `argocd-applicationset-controller`
- ServiceAccount for use by ApplicationSet controller, to access Argo CD resources
- Role granting RBAC access to needed resources, for ServiceAccount
- RoleBinding to bind the ServiceAccount and Role


### Install ApplicationSet and Argo CD together

You may install both the ApplicationSet controller and the latest stable Argo CD together, by creating a namespace and applying `manifests/install-with-argo-cd.yaml`:

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/applicationset/master/manifests/install-with-argo-cd.yaml
```

Once installed, follow the [Argo CD Getting Started](https://argoproj.github.io/argo-cd/getting_started/) to access Argo CD and log-in to the Web UI.

The ApplicationSet controller requires no additional setup.

### Customized install using Kustomize

To extend or customize the ApplicationSet controller installation, [Kustomize](https://kustomize.io/) may be used with the existing namespace install [manifests/namespace-install/kustomize.yaml](https://github.com/argoproj-labs/applicationset/blob/master/manifests/namespace-install/kustomization.yaml) file.


## Development builds

Development builds of the ApplicationSet controller can be installed by running the following command:
```
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/applicationset/master/manifests/install.yaml
```

You will need to ensure that Argo CD is already installed into the `argocd` namespace.

How it works:

- After each successful commit to *argoproj-labs/applicationset* `master` branch, a GitHub action will run that performs a container build/push to [`argoproj/argocd-applicationset:latest`](https://quay.io/repository/argoproj/argocd-applicationset?tab=tags )
- [Documentation for the `master`-branch-based developer builds](https://argocd-applicationset.readthedocs.io/en/master/)  is available from Read the Docs.

!!! warning

Development builds contain newer features and bug fixes, but are more likely to be unstable, as compared to release drivers.
See the `master` branch [Read the Docs](https://argocd-applicationset.readthedocs.io/en/master/) page for documentation on post-release features.


## Next Steps

Once your ApplicationSet controller is up and running, proceed to [Use Cases](Use-Cases.md) to learn more about the supported scenarios, or proceed directly to [Generators](Generators.md) to see example `ApplicationSet` resources. 
