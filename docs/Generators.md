# Generators

Generators are responsible for generating *parameters*, which are then rendered into the `template:` fields of the ApplicationSet resource.

As of this writing there are three generators: the List generator, the Cluster generator, and the Git generator. The Git generator contains two subtypes: File, and Directory.

## List Generator

The List generator generates parameters based on a fixed list of cluster name/URL values. In this example, we're targeting a local cluster named `engineering-dev`:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
 name: guestbook
spec:
 generators:
 - list:
     elements:
     - cluster: engineering-dev
       url: https://kubernetes.default.svc
#    - cluster: engineering-prod
#      url: https://kubernetes.default.svc
 template:
   metadata:
     name: '{{cluster}}-guestbook'
   spec:
     project: default
     source:
       repoURL: https://github.com/argoproj-labs/applicationset.git
       targetRevision: HEAD
       path: examples/list-generator/guestbook/{{cluster}}
     destination:
       server: '{{url}}'
       namespace: guestbook
```
(*The full example can be found [here](https://github.com/argoproj-labs/applicationset/tree/master/examples/list-generator).*)

The `url` and `cluster` fields are passed as parameters into the template in a straightforward fashion. If one wanted to add a second cluster, one could uncomment and apply the commented out elements, and the ApplicationSet controller would automatically target this new cluster.

**Note**: These clusters *must* already be defined within Argo CD, in order to generate applications for these values. The ApplicationSet controller does not create clusters within Argo CD (nor does it have the credentials to do so).

## Cluster Generator

In Argo CD, managed clusters [are stored within Secrets](https://argoproj.github.io/argo-cd/operator-manual/declarative-setup/#clusters) in the Argo CD namespace. The ApplicationSet controller uses those same Secret values to generate parameters to identify and target available clusters.

For each cluster registered with Argo CD, the Cluster generator produces a list of items found within the cluster secret. It automatically provides the following parameters as values to the Application template for each cluster:
- `name`
- `server`
- `metadata.labels.<key>` *(for each label in the Secret)*
- `metadata.annotations.<key>` *(for each annotation in the Secret)*

Within the [Argo CD cluster secret](https://argoproj.github.io/argo-cd/operator-manual/declarative-setup/#clusters) are data fields describing the cluster:
```yaml
kind: Secret
data:
  # Within Kubernetes these fields are actually encoded in Base64; they are decoded here for convenience. 
  # (They are likewise decoded when passed as parameters by the Cluster generator)
  config: "{'tlsClientConfig':{'insecure':false}}"
  name: "in-cluster2"
  server: "https://kubernetes.default.svc"
metadata:
  labels:
    argocd.argoproj.io/secret-type: cluster
# (...)
```

The Cluster generator will automatically identify clusters defined with Argo CD, and extract the cluster data as parameters:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: guestbook
spec:
  generators:
  - clusters: {} # Automatically use all clusters defined within Argo CD
  template:
    metadata:
      name: '{{name}}-guestbook' # 'name' field of the Secret
    spec:
      project: "default"
      source:
        repoURL: https://github.com/argoproj/argocd-example-apps/
        targetRevision: HEAD
        path: guestbook
      destination:
        server: '{{server}}' # 'server' field of the secret
        namespace: guestbook
```
(*The full example can be found [here](https://github.com/argoproj-labs/applicationset/tree/master/examples/cluster).*)

In this example, the cluster secret's `name` and `server` fields are used to populate the `Application` resource `name` and `server` (for the destination cluster).

### Label selector

A label selector may be used to narrow the scope of targeted clusters to only those matching a specific label:
```yaml
kind: ApplicationSet
metadata:
  name: guestbook
spec:
  generators:
  - clusters:
      selector:
        matchLabels:
          staging: true
  template:
  # (...)
```

This would match an Argo CD cluster secret containing:
```yaml
kind: Secret
data:
  # (... fields as above ...)
metadata:
  labels:
    argocd.argoproj.io/secret-type: cluster
    staging: "true"
# (...)
```


### Deploying to local cluster

The Cluster generator currently requires that an [Argo CD cluster secret exist](https://argoproj.github.io/argo-cd/operator-manual/declarative-setup/#clusters), in order to allow the ApplicationSet controller to create `Application` resources targeting that cluster.
    
However, if you are only using the 'local' cluster with Argo CD (ie the cluster upon which Argo CD is installed, when it is installed in cluster mode), then there will *not* be a Secret defined for your cluster: when using a local cluster, Argo CD does not create a cluster secret for it (or require one to exist).
    
Thus until [this ApplicationSet issue is resolved](https://github.com/argoproj-labs/applicationset/issues/35), if you want to use a local cluster with the Cluster generator then the workaround [here](https://github.com/argoproj-labs/applicationset/issues/35#issuecomment-769358837) can be used.


## Git Generator: Directories

The Git directory generator, one of two subtypes of the Git generator, generates parameters using the directory structure of a specified Git repository.

Suppose you have a Git repository with the following directory structure:
```
├── argo-workflows
│   ├── kustomization.yaml
│   └── namespace-install.yaml
└── prometheus-operator
    ├── Chart.yaml
    ├── README.md
    ├── requirements.yaml
    └── values.yaml
```

This reposistory contains two directories, one for each of the workloads to deploy: 
- an Argo Workflow controller kustomization YAML file
- a Prometheus Operator Helm chart

We can then deploy both workloads, using this example:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: cluster-addons
spec:
  generators:
  - git:
      repoURL: https://github.com/argoproj-labs/applicationset.git
      revision: HEAD
      directories:
      - path: examples/git-generator-directory/cluster-addons/*
  template:
    metadata:
      name: '{{path.basename}}'
    spec:
      project: default
      source:
        repoURL: https://github.com/argoproj-labs/applicationset.git
        targetRevision: HEAD
        path: '{{path}}'
      destination:
        server: https://kubernetes.default.svc
        namespace: '{{path.basename}}'
```
(*The full example can be found [here](https://github.com/argoproj-labs/applicationset/tree/master/examples/git-generator-directory).*)

The generator parameters are:
- `{{path}}`: The directory paths within the Git repository that match the `path` wildcard
- `{{path.basename}}`: For any directory path within the Git repository that matches the `path` wildcard, the right-most path name is extracted (eg `/directory/directory2` would produce `directory2`)

With this generator example, whenever a new Helm chart/Kustomize YAML/Application subfolder is added to the Git repository, the ApplicationSet controller will detect this change and automatically deploy the resulting manifests within new `Application` resources.

**Note**: As of this writing, only directories that are valid Argo CD applications (and specifically Helm, Kustomize, or Jsonnet applications) will be matched. This issue is [being tracked here](https://github.com/argoproj-labs/applicationset/issues/121).

As with other generators, clusters *must* already be defined within Argo CD, in order to generate Applications for them.

## Git Generator: Files

The Git file generator is the second subtype of the Git generator. The Git file generator generates parameters using the contents of JSON files found within a specified repository.

Suppose you have a Git repository with the following directory structure:
```
├── apps
│   └── guestbook
│       ├── guestbook-ui-deployment.yaml
│       ├── guestbook-ui-svc.yaml
│       └── kustomization.yaml
├── cluster-config
│   └── engineering
│       ├── dev
│       │   └── config.json
│       └── prod
│           └── config.json
└── git-generator-files.yaml
```

The folders are:
- `guestbook` is a simple guestbook application to deploy to the clusters. 
- `cluster-config` contains JSON files describing the individual engineering clusters: one for `dev` and one for `prod`.
- `git-generator-files.yaml` is the example `ApplicationSet` resource that deploys `guestbook` to the specified clusters.

The `config.json` files contain information describing the cluster (along with extra sample data):
```json
{
  "aws_account": "123456",
  "asset_id": "11223344",
  "cluster": {
    "owner": "cluster-admin@company.com",
    "name": "engineering-dev",
    "address": "http://1.2.3.4"
  }
}
```

The `config.json` files are automatically discovered by the Git generator, and the contents of those files are parsed and converted into template parameters. Here are the parameters generated that would be generated for the above JSON:
```
aws_account: 123456
asset_id: 11223344
cluster.owner: cluster-admin@company.com
cluster.name: engineering-dev
cluster.address: http://1.2.3.4
```


And the generated parameters for all discovered `config.json` files will be substituted into ApplicationSet template:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: guestbook
spec:
  generators:
  - git:
      repoURL: https://github.com/argoproj-labs/applicationset.git
      revision: HEAD
      files:
      - path: "examples/git-generator-files-discovery/cluster-config/**/config.json"
  template:
    metadata:
      name: '{{cluster.name}}-guestbook'
    spec:
      project: default
      source:
        repoURL: https://github.com/argoproj-labs/applicationset.git
        targetRevision: HEAD
        path: "examples/git-generator-files-discovery/apps/guestbook"
      destination:
        server: '{{cluster.address}}'
        namespace: guestbook
```
(*The full example can be found [here](https://github.com/argoproj-labs/applicationset/tree/master/examples/git-generator-files-discovery).*)

Any `config.json` files found under the `cluster-config` directory will be parameterized based on the `path` wildcard pattern specified. Within each file, JSON fields are flattened into key/value pairs, with this ApplicationSet example using the `cluster.address` as `cluster.name` parameters in the template.

**Note**: Support for parsing YAML files, in addition to JSON files, is planned. This work is [tracked here](https://github.com/argoproj-labs/applicationset/issues/106).

As with other generators, clusters *must* already be defined within Argo CD, in order to generate Applications for them.
