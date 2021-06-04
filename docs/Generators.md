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

The cluster selector also supports set-based requirements, as used by [several core Kubernetes resources](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#resources-that-support-set-based-requirements).

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
│   ├── kustomization.yaml
│   └── namespace-install.yaml
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

### Exclude directories

The Git directory generator also supports an `exclude` option in order to exclude directories in the repository from being scanned by the ApplicationSet controller:

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
      - path: examples/git-generator-directory/excludes/cluster-addons/*
      - exclude: true
        path: examples/git-generator-directory/excludes/cluster-addons/exclude-helm-guestbook
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
(*The full example can be found [here](https://github.com/argoproj-labs/applicationset/tree/master/examples/git-generator-directory/excludes).*)

This example excludes the `exclude-helm-guestbook` directory from the list of directories scanned for this `ApplictionSet` resource.

!!! note "Exclude rules have higher priority than include rules"

Every directory that matches at least one `exclude` pattern will always be excluded. Or, said another way, *exclude rules take precedence over include rules.*

As a corollary, the order of `path`s in the `directories` field does not change which directories are included/excluded (because, as above, exclude rules always take precedence over include rules). 

For example, with these directories:

```
.
└── d
    ├── e
    ├── f
    └── g
```
Say you want to include `/d/e`, but exclude `/d/f` and `/d/g`. This will *not* work:

```yaml
- path: /d/e
  exclude: false
- path: /d/*
  exclude: true
```
Why? Because the exclude `/d/*` exclude rule will take precedence over the `/d/e` include rule. When the `/d/e` path in the Git repository is processed by the ApplicationSet controller, the controller detects that at least one exclude rule is matched, and thus that directory should not be scanned.

You would instead need to do:

```yaml
- path: /d/*
- path: /d/f
  exclude: true
- path: /d/g
  exclude: true
```

Or, a shorter way (using [path.Match](https://golang.org/pkg/path/#Match) syntax) would be:

```yaml
- path: /d/*
- path: /d/[f|g]
  exclude: true
```

## Git Generator: Files

The Git file generator is the second subtype of the Git generator. The Git file generator generates parameters using the contents of JSON/YAML files found within a specified repository.

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
- `cluster-config` contains JSON/YAML files describing the individual engineering clusters: one for `dev` and one for `prod`.
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

As with other generators, clusters *must* already be defined within Argo CD, in order to generate Applications for them.

## SCM Provider Generator

The SCMProvider generator uses the API of an SCMaaS provider to discover repositories. This fits well with many repos following the same GitOps layout patterns such as microservices.

Support is currently limited to GitHub, PRs are welcome to add more SCM providers.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: myapps
spec:
  generators:
  - scmProvider:
      # Which protocol to clone using.
      cloneProtocol: ssh
      # See below for provider specific options.
      github:
        # ...
```

* `cloneProtocol`: Which protocol to use for the SCM URL. Default is provider-specific but ssh if possible. Not all providers necessarily support all protocols, see provider documentation below for available options.

### GitHub

The GitHub mode uses the GitHub API to scan and organization in either github.com or GitHub Enterprise.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: myapps
spec:
  generators:
  - scmProvider:
      github:
        # The GitHub organization to scan.
        organization: myorg
        # For GitHub Enterprise:
        api: https://git.example.com/
        # If true, scan every branch of every repository. If false, scan only the default branch. Defaults to false.
        allBranches: true
        # Reference to a Secret containing an access token. (optional)
        tokenRef:
          secretName: github-token
          key: token
  template:
  # ...
```

* `organization`: Required name of the GitHub organization to scan. If you have multiple orgs, use multiple generators.
* `api`: If using GitHub Enterprise, the URL to access it.
* `allBranches`: By default (false) the template will only be evaluated for the default branch of each repo. If this is true, every branch of every repository will be passed to the filters. If using this flag, you likely want to use a `branchMatch` filter.
* `tokenRef`: A Secret name and key containing the GitHub access token to use for requests. If not specified, will make anonymous requests which have a lower rate limit and can only see public repositories.

For label filtering, the repository topics are used.

Available clone protocols are `ssh` and `https`.

### Filters

Filters allow selecting which repositories to generate for. Each filter can declare one or more conditions, all of which must pass. If multiple filters are present, any can match for a repository to be included. If no filters are specified, all repositories will be processed.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: myapps
spec:
  generators:
  - scmProvider:
      filters:
      # Include any repository starting with "myapp" AND including a Kustomize config AND labeled with "deploy-ok" ...
      - repositoryMatch: ^myapp
        pathsExist: [kubernetes/kustomization.yaml]
        labelMatch: deploy-ok
      # ... OR any repository starting with "otherapp" AND a Helm folder.
      - repositoryMatch: ^otherapp
        pathsExist: [helm]
  template:
  # ...
```

* `repositoryMatch`: A regexp matched against the repository name.
* `pathsExist`: An array of paths within the repository that must exist. Can be a file or directory, but do not include the trailing `/` for directories.
* `labelMatch`: A regexp matched against repository labels. If any label matches, the repository is included.
* `branchMatch`: A regexp matched against branch names.

### Template

As with all generators, several keys are available for replacement in the generated application.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: myapps
spec:
  generators:
  - scmProvider:
    # ...
  template:
    metadata:
      name: '{{ repository }}'
    spec:
      source:
        repoURL: '{{ url }}'
        targetRevision: '{{ branch }}'
        path: kubernetes/
      project: default
      destination:
        server: https://kubernetes.default.svc
        namespace: default
```

* `organization`: The name of the organization the repository is in.
* `repository`: The name of the repository.
* `url`: The clone URL for the repository.
* `branch`: The default branch of the repository.

## Matrix Generator
The matrix generator combines two other generators by multiplying the parameters of them.

### Use Case Example
Imagine we have two clusters: 
* staging (@ https://1.2.3.4)
* production (@ https://2.4.6.8)

And our application yamls are defined in a git repository:
* argo workflows controller (examples/git-generator-directory/cluster-addons/argo-workflows)
* prometheus operator (/examples/git-generator-directory/cluster-addons/prometheus-operator)

Our goal is to deploy both application in both clusters. 
For that we will use the matrix generator, with the git and the cluster as inner generators:

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

First the git directory generator which will produce:
```yaml
    - path: /examples/git-generator-directory/cluster-addons/argo-workflows
      path.basename: argo-workflows
      
    - path: /examples/git-generator-directory/cluster-addons/prometheus-operator
      path.basename: prometheus-operator
```
Second the cluster generator which will produce:
```yaml
    - name: staging
      server: https://1.2.3.4
      
    - name: production
      server: https://2.4.6.8
```
The matrix generators will combine both output and produce:
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

### Restrictions
1. The matrix generator currently supports only two inner generators.
2. The inner generators should only have one generator:
   This is not a valid example:
```yaml
- matrix:
    generators:
     - list:...
       git: ...
```
3. The matrix generator ignores templates of the inner generators
```yaml
- matrix:
    generators:
      - list:
          elements: []
          template: # Ignored
       
```

## Cluster List Resource Generator

The cluster list resource generates a list of Argo CD clusters. This is done using [duck-typing](https://pkg.go.dev/knative.dev/pkg/apis/duck), which does not require knowledge of the full shape of the referenced kubernetes resource. The following is an example of the new ApplicationSet generator:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
 name: guestbook
spec:
 generators:
 - clusterDecisionResource:
    configMapRef: my-configmap  # ConfigMap with GVK information for the duck type resource
    name: quak                  # The name of the resource
    requeueAfterSeconds: 60     # OPTIONAL: Checks for changes every 60sec (default 3min)
 template:
   metadata:
     name: '{{name}}-guestbook'
   spec:
      project: "default"
      source:
        repoURL: https://github.com/argoproj/argocd-example-apps/
        targetRevision: HEAD
        path: guestbook
      destination:
        server: '{{clusterName}}' # 'server' field of the secret
        namespace: guestbook
```
The `quak` resource, referenced by the ApplicationSet `clusterDecisionResource` generator:
```yaml
apiVersion: mallard.io/v1beta1
kind: Duck
metadata:
  name: quak
spec: {}
status:
  decisions:     # Duck-typing ignores all other aspects of the resource except the "decisions" list
  - clusterName: cluster-01
  - clusterName: cluster-02
```
The ApplicationSet references a ConfigMap that defines the resource to be used in this duck-typing. Only one ConfigMap is required per ArgoCD instance, to identify a resource. You can support multiple resource types by creating a ConfigMap for each.
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-configmap
data:
  apiVersion: mallard.io/v1beta1  # apiVersion of the target resource
  kind: ducks                     # kind of the target resource
  statusListKey: decisions        # status key name that holds the list of ArgoCD clusters
  matchKey: clusterName           # The key in the status list whose value is the cluster name found in ArgoCD
```

(*The full example can be found [here](https://github.com/argoproj-labs/applicationset/tree/master/examples/clusterDecisionResource).*)

This example leverages the cluster management capabilities of the [open-cluster-management.io community](https://open-cluster-management.io/). By creating a ConfigMap with the GVK for the open-cluter-management.io Placement rule, your ApplicationSet can provision to different clusters in a number of novel ways. One example is to have the ApplicationSet maintain only two ArgoCD Applicaitons across 3 or more clusters. Then as maintenance or outages occur, the ApplicationSet will always maintain two Applications, moving the application to available clusters under the Placement rule's direction. 

### How it works
The ApplicationSet needs to be created in the ArgoCD namespace, placing the ConfigMap in the same namespace allows the ClusterDecisionResource generator to read it. The ConfigMap stores the GVK information as well as the status key definitions.  In the open-cluster-management example, the ApplicationSet generator will read the kind `placementrules` with an apiVersion of `apps.open-cluster-management.io/v1`. It will attempt to extract the **list** of clusters from the key `decisions`. It then validates the actual cluster name as defined in ArgoCD against the **value** from the key `clusterName` in each of the elements in the list.

The ClusterDecisionResource generator passes the 'name', 'server' and any other key/value in the duck-type resource's status list as parameters into the ApplicationSet template. In this example, the decision array contained an additional key `clusterName`, which is now available to the ApplicationSet template.

!!! note "Clusters listed as `Status.Decisions` must be predefined in Argo CD"
    The cluster names listed in the `Status.Decisions` *must* be defined within Argo CD, in order to generate applications for these values. The ApplicationSet controller does not create clusters within Argo CD.

    The Default Cluster list key is `clusters`.