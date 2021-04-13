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

The List generator passes the `url` and `cluster` fields as parameters into the template. In this example, if one wanted to add a second cluster, we could uncomment the second cluster element and the ApplicationSet controller would automatically target it with the defined application.

!!! note "Clusters must be predefined in Argo CD"
    These clusters *must* already be defined within Argo CD, in order to generate applications for these values. The ApplicationSet controller does not create clusters within Argo CD (for instance, it does not have the credentials to do so).

## Cluster Generator

In Argo CD, managed clusters [are stored within Secrets](https://argoproj.github.io/argo-cd/operator-manual/declarative-setup/#clusters) in the Argo CD namespace. The ApplicationSet controller uses those same Secrets to generate parameters to identify and target available clusters.

For each cluster registered with Argo CD, the Cluster generator produces parameters based on the list of items found within the cluster secret. 

It automatically provides the following parameter values to the Application template for each cluster:

- `name`
- `server`
- `metadata.labels.<key>` *(for each label in the Secret)*
- `metadata.annotations.<key>` *(for each annotation in the Secret)*

Within [Argo CD cluster Secrets](https://argoproj.github.io/argo-cd/operator-manual/declarative-setup/#clusters) are data fields describing the cluster:
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

In this example, the cluster secret's `name` and `server` fields are used to populate the `Application` resource `name` and `server` (which are then used to target that same cluster).

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

### Deploying to the local cluster

In Argo CD, the 'local cluster' is the cluster upon which Argo CD (and the ApplicationSet controller) is installed. This is to distinguish it from 'remote clusters', which are those that are added to Argo CD [declaratively](https://argoproj.github.io/argo-cd/operator-manual/declarative-setup/#clusters) or via the [Argo CD CLI](https://argoproj.github.io/argo-cd/getting_started/#5-register-a-cluster-to-deploy-apps-to-optional).
 
The cluster generator will automatically target both local and non-local clusters, for every cluster that matches the cluster selector.

If you wish to target only remote clusters with your Applications (e.g. you want to exclude the local cluster), then use a cluster selector with labels, for example:
```yaml
spec:
  generators:
  - clusters:
      selector:
        matchLabels:
          argocd.argoproj.io/secret-type: cluster
```

This selector will not match the default local cluster, since the default local cluster does not have a Secret (and thus does not have the `argocd.argoproj.io/secret-type` label on that secret). Any cluster selector that selects on that label will automatically exclude the default local cluster.

However, if you do wish to target both local and non-local clusters, while also using label matching, you can create a secret for the local cluster within the Argo CD web UI:

1. Within the Argo CD web UI, select *Settings*, then *Clusters*.
2. Select your local cluster, usually named `in-cluster`.
3. Click the *Edit* button, and change the the *NAME* of the cluster to another value, for example `in-cluster-local`. Any other value here is fine. 
4. Leave all other fields unchanged.
5. Click *Save*.

These steps might seem counterintuitive, but the act of changing one of the default values for the local cluster causes the Argo CD Web UI to create a new secret for this cluster. In the Argo CD namespace, you should now see a Secret resource named `cluster-(cluster suffix)` with label `argocd.argoproj.io/secret-type": "cluster"`. You may also create a local [cluster secret declaratively](https://argoproj.github.io/argo-cd/operator-manual/declarative-setup/#clusters), or with the CLI using `argocd cluster add "(context name)" --in-cluster`, rather than through the Web UI.

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

We can deploy both workloads, using this example:
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

- `{{path}}`: The directory paths within the Git repository that match the `path` wildcard.
- `{{path.basename}}`: For any directory path within the Git repository that matches the `path` wildcard, the right-most path name is extracted (e.g. `/directory/directory2` would produce `directory2`).

Whenever a new Helm chart/Kustomize YAML/Application/plain subfolder is added to the Git repository, the ApplicationSet controller will detect this change and automatically deploy the resulting manifests within new `Application` resources.

As with other generators, clusters *must* already be defined within Argo CD, in order to generate Applications for them.

The generator also supports `exclude` option in order to exclude directories in the repository from being created by the applicationset;

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
      - exclude: true
        path: examples/git-generator-directory/cluster-addons/exclude-helm-guestbook
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

- `guestbook` contains the Kubernetes resources for a simple guestbook application
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
    "address": "https://1.2.3.4"
  }
}
```

Git commits containing changes to the `config.json` files are automatically discovered by the Git generator, and the contents of those files are parsed and converted into template parameters. Here are the parameters generated for the above JSON:
```text
aws_account: 123456
asset_id: 11223344
cluster.owner: cluster-admin@company.com
cluster.name: engineering-dev
cluster.address: https://1.2.3.4
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

Any `config.json` files found under the `cluster-config` directory will be parameterized based on the `path` wildcard pattern specified. Within each file JSON fields are flattened into key/value pairs, with this ApplicationSet example using the `cluster.address` as `cluster.name` parameters in the template.

!!! note "Supported file formats"
    Only JSON file parsing is currently supported. The work to add support for YAML files is [tracked here](https://github.com/argoproj-labs/applicationset/issues/106).

As with other generators, clusters *must* already be defined within Argo CD, in order to generate Applications for them.
