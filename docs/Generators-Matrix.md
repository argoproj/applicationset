# Matrix Generator

The Matrix generator combines two other generators by multiplying the parameters of them.

## Use Case Example

Imagine we have two clusters: 

- staging (at `https://1.2.3.4`)
- production (at `https://2.4.6.8`)

And our application YAMLs are defined in a Git repository:

- Argo Workflows controller (examples/git-generator-directory/cluster-addons/argo-workflows)
- Prometheus operator (/examples/git-generator-directory/cluster-addons/prometheus-operator)

Our goal is to deploy both application in both clusters. 
For that we will use the Matrix generator, with the Git and the Cluster as inner generators:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: cluster-git
spec:
  generators:
    - matrix:
        generators:
            - git:
                repoURL: https://github.com/argoproj-labs/applicationset.git
                revision: HEAD
                directories:
                  - path: examples/matrix/cluster-addons/*
            - clusters:
                selector:
                  matchLabels:
                    argocd.argoproj.io/secret-type: cluster
  template:
    metadata:
      name: '{{path.basename}}-{{name}}'
    spec:
      project: '{{metadata.labels.environment}}'
      source:
        repoURL: https://github.com/argoproj-labs/applicationset.git
        targetRevision: HEAD
        path: '{{path}}'
      destination:
        server: '{{server}}'
        namespace: '{{path.basename}}'
```

First, the Git directory generator will produce:
```yaml
    - path: /examples/git-generator-directory/cluster-addons/argo-workflows
      path.basename: argo-workflows
      
    - path: /examples/git-generator-directory/cluster-addons/prometheus-operator
      path.basename: prometheus-operator
```
Second, the Cluster generator will produce:
```yaml
    - name: staging
      server: https://1.2.3.4
      
    - name: production
      server: https://2.4.6.8
```
The Matrix generator will combine both outputs and produce:
```yaml
    - name: staging
      server: https://1.2.3.4
      path: /examples/git-generator-directory/cluster-addons/argo-workflows
      path.basename: argo-workflows

    - name: staging
      server: https://1.2.3.4
      path: /examples/git-generator-directory/cluster-addons/prometheus-operator
      path.basename: prometheus-operator

    - name: production
      server: https://2.4.6.8
      path: /examples/git-generator-directory/cluster-addons/argo-workflows
      path.basename: argo-workflows

    - name: production
      server: https://2.4.6.8      
      path: /examples/git-generator-directory/cluster-addons/prometheus-operator
      path.basename: prometheus-operator

```
(*The full example can be found [here](https://github.com/argoproj-labs/applicationset/tree/master/examples/matrix).*)

## Restrictions

1. The Matrix generator currently supports only two inner generators.
2. The inner generators should only have one generator:
   Eg this is not valid:
```yaml
- matrix:
    generators:
     - list:...
       git: ...
```
3. The Matrix generator ignores templates of the inner generators
```yaml
- matrix:
    generators:
      - list:
          elements: []
          template: # Ignored
```
