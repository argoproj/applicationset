# Changelog

# v0.4.1

## Contributors
- Alexander Matyushentsev (@alexmt)

## New in this release

This is a bug-fix release that fixes an issue with Git credential acquisition by the ApplicationSet controller in the v0.4.0 ApplicationSet controller and Argo CD v2.3.0 (due to packaging ApplicationSet controller v0.4.0).
- [ArgoCD v2.3.0 - ARGOCD_GIT_ASKPASS_NONCE is not set](https://github.com/argoproj/argo-cd/issues/8716)

# v0.4.0

## Contributors

Thanks to all the folks who have contributed to the ApplicationSet controller since our last release. 

- Michael Crenshaw (@crenshaw-dev)
- Ricardo Rosales (@missingcharacter)
- Ishita Sequeira  (@ishitasequeira)
- stempher (@stempher)
- Jonathan West (@jgwest)
- Hüseyin Celal Öner (@hcelaloner)
- William Tam (@wtam2018)
- Marco Kilchhofer (@mkilchhofer)
- Chetan Banavikalmutt (@chetan-rns)
- Ahmed AbouZaid (@aabouzaid)
- Matthias Lisin  (@ml-)

Want to join us for our next release? Check out the project repository (https://github.com/argoproj/applicationset) or visit us on #argo-cd-appset on Slack (https://argoproj.github.io/community/join-slack/).

## New in this release

### ApplicationSet controller is now integrated with Argo CD install

The ApplicationSet controller is now installed by default with the latest release of Argo CD. This means that users of Argo CD now get all the benefits of ApplicationSets, without the requirement of a standalone install.

Contributed by [@ishitasequeira](https://github.com/argoproj/applicationset/pull/455) and [@jgwest](https://github.com/argoproj/applicationset/pull/470).

### Git generator: Add support for extraction of components of paths

The Git generator now supports the extraction of individual components of the path, with the new `path[n]` parameter: 
- `{{path[n]}}`: The path to the matching configuration file within the Git repository, split into array elements (`n` - array index). 
- For example, for a path of `/clusters/clusterA`, the individual components can be extracted like so: `path[0]: clusters`, `path[1]: clusterA`

Contributed by [@stempher](https://github.com/argoproj/applicationset/pull/389).

### Git generator: Sanitize basename param by replacing unsupported characters

When using the Git generator, with a basename name param that contains an unsupported character, you may now use the `{{path.basenameNormalized}}` parameter to normalize these resources. This prevents rendering invalid Kubernetes resources with names like `my_cluster-app1`, and instead would convert them to `my-cluster-app1`.

Contributed by [@missingcharacter](https://github.com/argoproj/applicationset/pull/436).

### Make webhook address configurable 

When using a webhook, the address of the webhook can now be configured using the `--webhook-addr` parameter on the `argocd-applicationset` controller.

Example:
```
./dist/argocd-applicationset --webhook-addr=":9999" --logformat=json
```

Contributed by [@chetan-rns](https://github.com/argoproj/applicationset/pull/450).



#### Fixes / Chores

- ApplicationSet CRD size reduction, by removing validation (CRD defn) of nested merge/matrix generator (#463, contributed by @jgwest)
- Reap zombie processes in argocd-applicationset-controller pod using tini (#453, contributed by @hcelaloner)
- Log all validation errors (#439, contributed by @crenshaw-dev)
- Set applicationset-controller containerPort name (#444, contributed by @aabouzaid)
- Append missing s to matchExpression (#449, contributed by @ml-)
- Set controller logger if we don't use JSON format (#451, contributed by @mkilchhofer)
- Remove hardcoded namespace from manifests (#474, contributed by @ishitasequeira)
- Fix docs typo (#493 and #481, contributed by @crenshaw-dev)

#### Test/infrastructure improvements:
- E2E tests should use application-controller serviceaccount, rather than applicationset-controller serviceaccount (#434, contributed by @jgwest)
- Add GitHub action to run E2E tests against nightly Argo CD, w/ ApplicationSet master branch (#470, contributed by @jgwest)


# v0.3.0

I am happy to announce the latest release of the Argo CD ApplicationSet controller, v0.3.0. Many new features were contributed as part of this release, including two new generators, improved error reporting and handling, support for webhook-based refresh trigger, plus doc updates, usability improvements, stability fixes, and more. 

You can learn more about this release from the [ApplicationSet documentation](https://argocd-applicationset.readthedocs.io), or check out the project repository [and learn how you can contribute](https://github.com/argoproj-labs/applicationset/).

## Contributors

Thanks to all the folks who have contributed to the ApplicationSet controller since our last release. 
- Shunya Murata (@shmurata)
- Michael Crenshaw (@crenshaw-dev)
- Jonathan West (@jgwest)
- Ishita Sequeira (@ishitasequeira)
- Chetan Banavikalmutt (@chetan-rns)
- Alexander Matyushentsev (@alexmt)
- Shiv Jha-Mathur (@shivjm)
- Subhash Chandra (@TMaYaD)
- William Tam (@wtam2018)
- Benoit Gaillard (@benoitg31) 
- Michal Barecki (@mbarecki)
- Guillaume Dupin (@yogeek)
- Krzysztof Dąbrowski (@krzysdabro)
- Olve S. Hansen (@olvesh)
- Dewan Ishtiaque Ahmed (@dewan-ahmed)
- Diego Pomares (@DiegoPomares)

Want to join us for our next release? Check out the project repository (https://github.com/argoproj-labs/applicationset) or visit us on #argo-cd-appset on Slack (https://argoproj.github.io/community/join-slack/).

## New in this release

### New generator: Pull Request generator

With ApplicationSet v0.3.0, a new Pull Request generator has been contributed which uses the API of an SCMaaS provider (e.g. GitHub) to automatically discover open pull requests within an repository. This fits well with users that wish to construct a test environment based on an open pull request.

In this example, we will create an Argo CD `Application` resource for each open pull request:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: myapps
spec:
  generators:
  - pullRequest:
      github:
        # The GitHub organization or user.
        owner: myorg
        # The Github repository
        repo: myrepository
        # For GitHub Enterprise (optional)
        api: https://git.example.com/
        # Reference to a Secret containing an access token. (optional)
        tokenRef:
          secretName: github-token
          key: token
        # Labels is used to filter the PRs that you want to target. (optional)
        labels:
        - preview
  template:
  # (template the Application using PR generator params)...
```  

To learn more, check out the [Pull Request generator documentation](https://argocd-applicationset.readthedocs.io/en/master/Generators-Pull-Request/) for details. Contributed by [@shmurata](https://github.com/argoproj-labs/applicationset/pull/366/).

### New generator: Merge generator

Also new in this release is the Merge generator, which is useful when you want to selectively override the parameters generated by one generator, with those generated by another.

In this example, we first gather the list of clusters from Argo CD, then we 'patch' only those clusters with label `use-kakfa: false`, and finally we enable redis on a specfic cluster:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: cluster-git
spec:
  generators:
    # merge 'parent' generator
    - merge:
        mergeKeys:
          - server
        generators:
          # Generate parameters for all Argo CD clusters
          - clusters:
              values:
                kafka: 'true'
                redis: 'false'
          # For clusters with a specific label, enable Kafka.
          - clusters:
              selector:
                matchLabels:
                  use-kafka: 'false'
              values:
                kafka: 'false'
          # For a specific cluster, enable Redis.
          - list:
              elements: 
                - server: https://2.4.6.8
                  values.redis: 'true'
```

See the [Merge generator documentation](https://argocd-applicationset.readthedocs.io/en/master/Generators-Merge/) for a full example, and for details on generator behaviour. Contributed by [@crenshaw-dev](https://github.com/argoproj-labs/applicationset/pull/404).

### Report error conditions/status for ApplicationSet CR
   
When the user-provided generator/template produce invalid Argo CD Applications, the `ApplicationSet` resource's status field will now report errors (or the lack thereof). Here is an example of the new status conditions:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: myapps
spec:
  generators: # (...)
  template: # (...)
status:
  conditions:
  - lastTransitionTime: "2021-11-23T05:47:08Z"
    type: ErrorOccurred
    status: "False"
    reason: ApplicationSetUpToDate
    message: Successfully generated parameters for all Applications
  - lastTransitionTime: "2021-11-23T05:47:08Z"
    type: ParametersGenerated
    message: Successfully generated parameters for all Applications
    reason: ParametersGenerated
    status: "True"
  - lastTransitionTime: "2021-11-23T05:47:08Z"
    type: ResourcesUpToDate
    status: "True"
    reason: ApplicationSetUpToDate
    message: ApplicationSet up to date
```

On parameter generation failure or templating failure, those errors will be reported under the appropriate conditions. Contributed by [@ishitasequeira](https://github.com/argoproj-labs/applicationset/pull/370).


### Git Generator: Refresh ApplicationSet resource with Git generator using webhook

This feature adds support for refreshing ApplicationSets via a GitHub webhook trigger. It exposes a service which listens for incoming webhook payloads, and once received triggers the ApplicationSet controller to regenerate resources. In contrast, with the previous release, the ApplicationSet controller only supported polling the Git repository used by the Git generator every 3 mins (but this is at least customizable).

See the [webhook documentation](https://argocd-applicationset.readthedocs.io/en/master/Generators-Git/#webhook-configuration) for details. Contributed by [@chetan-rns](https://github.com/argoproj-labs/applicationset/pull/341).

This contribution also adds general support for webhooks, which is used by the Pull Request generator webhook code, below.

### Gracefully handle application validation errors

This feature changes how the ApplicationSet controller handles ApplicationSets that generate invalid `Application`s. Previously, if at least one Application in the ApplicationSet was invalid, the controller would refuse to proceed further and would skip _all_ Application processing (i.e. it would 'fail fast'). Now, the controller will process *valid* Applications, and only skip *invalid* Applications (logging information about them to the console).

Contributed by [@alexmt](https://github.com/argoproj-labs/applicationset/pull/372).

### Pull Request generator: Support for webhooks

When using a Pull Request generator, the ApplicationSet controller polls every `requeueAfterSeconds` interval (defaulting to every 30 minutes) to detect changes. To eliminate this delay from polling, the ApplicationSet webhook server can be configured to receive webhook events, which will refresh the parameters generated by the Pull Request generator, and thus the corresponding `Application` resources.

More information on configuring webhooks with the Pull Request generator is available from the [Pull Request generator documentation](https://argocd-applicationset.readthedocs.io/en/master/Generators-Pull-Request/#webhook-configuration). Contributed by [@shmurata](https://github.com/argoproj-labs/applicationset/pull/417).

### Support `-logformat=json` as parameter to applicationset-controller

This feature adds a new `--logformat=json` parameter to the applicationset-controller, which switches the logging output of the ApplicationSet controller to JSON. Contributed by [@shivjm](https://github.com/argoproj-labs/applicationset/pull/373).

### SCM Generator: Provide SHA for latest commit on a branch in variables (#307)

This feature adds SHA to the list of parameters exposed by the SCM Generator, with the SHA parameter representing the latest commit. Contributed by [@TMaYaD](https://github.com/argoproj-labs/applicationset/pull/307).

### Improve Git files generator performance (#355)

The Git files generator was consuming too much time (and driving up Git requests) due to inadvertently executing 'git fetch/git checkout' for each discovered file within the repository. With ApplicationSet v0.3.0, that has improved such that we will now issue a Git checkout/fetch repo once per refresh. Contributed by [@alexmt](https://github.com/argoproj-labs/applicationset/pull/355).

### Fixes, test fixes, infrastructure improvements, and documentation updates

#### Fixes
- Fix: new variable for the normalized version of name field ([#390](https://github.com/argoproj-labs/applicationset/pull/390), contributed by @chetan-rns)
- Fixes GitLab RepoHasPath error handling ([#423](https://github.com/argoproj-labs/applicationset/pull/423), contributed by @benoitg31)


#### Test/infrastructure improvements:
- Investigate Argo CD deletion failure messages when running ApplicationSet E2E tests in on-cluster configuration ([#392](https://github.com/argoproj-labs/applicationset/pull/392), contributed by @jgwest)
- Update master branch VERSION file and metadata, and pull up release changes from 0.2.0 ([#343](https://github.com/argoproj-labs/applicationset/pull/343), contributed by @jgwest)
- Skip E2E tests that require GitHub token, if not specified ([#380](https://github.com/argoproj-labs/applicationset/pull/380), contributed by @jgwest)
- API rate limit error in image publish action ([#368](https://github.com/argoproj-labs/applicationset/pull/368), contributed by @jgwest)
- Disable SCM Provider Unit tests on PRs ([#337](https://github.com/argoproj-labs/applicationset/pull/337), contributed by @jgwest)
- Fix lint-docs ([#411](https://github.com/argoproj-labs/applicationset/pull/411), contributed by @crenshaw-dev)
- Fix indentation in example ([#360](https://github.com/argoproj-labs/applicationset/pull/360), contributed by @DiegoPomares)
- Adding required 'version' field for Helm Charts ([#332](https://github.com/argoproj-labs/applicationset/pull/332), contributed by @dewan-ahmed)
- Adopt latest Argo CD dependencies, in preparation for next release ([#410](https://github.com/argoproj-labs/applicationset/pull/410), contributed by @jgwest)


#### Doc updates
- Update release process docs and include release checklist in docs ([#365](https://github.com/argoproj-labs/applicationset/pull/365), contributed by @jgwest)
- Fix includeSubgroups reference name ([#357](https://github.com/argoproj-labs/applicationset/pull/357), contributed by @yogeek)
- Add missing brace ([#349](https://github.com/argoproj-labs/applicationset/pull/349), contributed by @krzysdabro)
- Fix Git Generator Files path example in docs ([#408](https://github.com/argoproj-labs/applicationset/pull/408), contributed by @mbarecki)
- Corrected wrong info about path and path.basename ([#412](https://github.com/argoproj-labs/applicationset/pull/412), contributed by @olvesh)



## Upgrade Notes

When moving from ApplicationSet v0.1.0/v0.2.0, to v0.3.0, there are two behaviour changes to be aware of.

#### Cluster generator: `{{name}}` parameter value will no longer be normalized, but existing normalization behaviour is preserved in a new `{{nameNormalized}}` parameter

The Cluster generator `{{name}}` parameter has now reverted to its original behaviour: the cluster name within Argo CD will no longer be [normalized](https://github.com/argoproj-labs/applicationset/blob/11f1fe893b019c9a530865fa83ee78b16af2c090/pkg/generators/cluster.go#L168). The `{{name}}` parameter generated by the Cluster generator within the ApplicationSet will now be passed unmodified to the ApplicationSet template. 

A new parameter, `{{nameNormalized}}` has been introduced which preserves the 0.2.0 behaviour. This allows you to choose which behaviour you wish to use in your ApplicationSet, based on the context in which it is used: either using the parameter as defined, or in a normalized form (which allows it to be used in the `name` field of an `Application` resource.)

If your Argo CD cluster names are already valid, no change is required. Otherwise, to preserve the v0.2.0 behaviour of your ApplicationSet, replace `{{name}}` with `{{nameNormalized}}` within your ApplicationSet template. 

More information about this change is [available from the issue](https://github.com/argoproj-labs/applicationset/pull/390).


#### If an ApplicationSet contains an invalid generated Application, the valid generated Applications will still be processed

The responsibility of the ApplicationSet controller is to convert an `ApplicationSet` resource into one or more `Application` resources. However, with the previous releases, if at least one of the generated `Application` resources was invalid (e.g. it failed the internal validation logic), none of the generated Applications would be processed (they would not be neither created nor modified).

With the latest ApplicationSet release, if a generator generates invalid Applications, those invalid generated Applications will still be skipped, **but** the valid generated Applications will now be processed (created/modified).

Thus no `ApplicationSet` resource changes are required by this new behaviour, but it is worth keeping in mind that your ApplicationSets which were previously blocked by a failing Application may no longer be blocked. This change might cause valid Applications to now be created/modified, whereas previously they were prevented from being processed.

More information about this change is [available from the issue](https://github.com/argoproj-labs/applicationset/pull/372).


## Installation

The ApplicationSet controller must be installed into the same namespace as the Argo CD it is targeting:
```
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/applicationset/v0.3.0/manifests/install.yaml
```

Once installed, the ApplicationSet controller requires no additional setup. You can learn more about ApplicationSet controller installation from the [Getting Started](https://argocd-applicationset.readthedocs.io/en/v0.3.0/Getting-Started/) page.



# v0.2.0

I am happy to announce the second release of the Argo CD ApplicationSet controller, v0.2.0. Many new features were contributed as part of this release, including support for combining generator parameters, support for building Argo CD Applications based on GitHub/GitLab organizations, and support for using custom resources to select clusters, plus oft requested improvements to existing generators, and of course doc updates, usability improvements, stability fixes, and more. 

You can learn more about this from the [ApplicationSet documentation](https://argocd-applicationset.readthedocs.io) or check out the source [and learn how you can contribute](https://github.com/argoproj-labs/applicationset/).

## Contributors

Many many thanks to all the folks who have contributed to the ApplicationSet controller over the past few months. These many contributions, both big and small, general and specific, help to bring a more featureful and polished experience to Argo CD users. We could not do this without all of you!
- Omer Kahani (@OmerKahani)
- Joshua Packer (@jnpacker)
- Jonathan West (@jgwest)
- Noah Kantrowitz (@coderanger)
- Lior Lieberman (@LiorLieberman)
- Michael Matur (@mmatur)
- TJ Miller (@teejaded)
- John Thompson (@empath-nirvana)
- Ishita Sequeira (@ishitasequeira)
- Chetan Banavikalmutt (@chetan-rns)
- William Tam (@wtam2018)
- John Watson (@dctrwatson)
- Tencho Tenev (@tenevdev)
- Ryan Umstead (@rumstead)
- Stephan Auerhahn(@palp)
- Christian Hernandez (@christianh814)
- Mohit Kumar Sharma (@mksha)
- Vivien Fricadel (@vivienfricadelamadeus)
- Gareth Western (@gdubya)
- William Jeffries (@williamcodes)
- Mehran Poursadeghi (@mehran-prs)

Want to join us for our next release? Check out the project repository (https://github.com/argoproj-labs/applicationset) or visit us on #argo-cd-appset on Slack (https://argoproj.github.io/community/join-slack/).

## New in this Release

### Matrix generator

The Matrix generator is a new generator that combines the parameters generated by two child generators, iterating through every combination of each generator's set:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: cluster-git
spec:
  generators:
    - matrix: # 'Parent' Matrix Generator
        generators:
          - git: # 'Child' generator A
              repoURL: https://github.com/argoproj-labs/applicationset.git
              revision: HEAD
              directories:
                - path: examples/matrix/cluster-addons/*
          - clusters: # 'Child' generator B
              selector:
                matchLabels:
                  argocd.argoproj.io/secret-type: cluster
  template:
  # (...)
```

The parameters generated by a Git generator (looking for directories within a Git Repository), and by a Cluster generator (looking for Argo CD-managed clusters), are combined: this will generate Argo CD Applications that target each of the Git directories (containing Kubernetes manifests), for each cluster managed by Argo CD.

See the [Matrix generator documentation](https://argocd-applicationset.readthedocs.io/en/master/Generators-Matrix/) for an in-depth example of how generator parameters are combined. Contributed by [@OmerKahani](https://github.com/argoproj-labs/applicationset/pull/205).


### SCM Provider generator

The SCM Provider generator is a new generator that utilizes the API of GitHub/GitLab to automatically discover repositories within an organization, allowing the repository values to be used for targeting generated Argo CD Applications. This fits well with GitOps layout patterns that split microservices across many repositories, rather than those patterns that stick to a single repository (which can be handled by other ApplicationSet generators).

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: myapps
spec:
  generators:
  - scmProvider:
      github:
        organization: myorg         # The GitHub organization to scan.
        tokenRef: { } # Reference to a Secret containing an access token. (optional)
  template:
  # ...
```

The ApplicationSet controller will then scan the provided GitHub/GitLab organization and produce template parameters for each discovered repository/branch, which may be used to generate Argo CD Applications for each of these repositories.

See the [SCM Provider generator](https://argocd-applicationset.readthedocs.io/en/master/Generators-SCM-Provider/) documentation for more information. Initial implementation and GitHub support contributed by [@coderanger](https://github.com/argoproj-labs/applicationset/pull/209), and GitLab support contributed by [@empath-nirvana](https://github.com/argoproj-labs/applicationset/pull/283).


### Cluster Decision Resource generator

The Cluster Decision Resource generator is a new generator that generates a list of Argo CD clusters based on the contents of an external custom resource (CR), with that custom resource managed by an external controller. With the Cluster Decision Resource generator, you may 'outsource' the logic (of which clusters to target) to third party controllers/CRs, such as the [Open Cluster Management's Placements](https://open-cluster-management.io/concepts/placement/). 

This is handled seamlessly using [duck-typing](https://knative.dev/docs/developer/concepts/duck-typing/), which does not require knowledge of the full shape of the referenced Kubernetes resource. The following is an example of a cluster-decision-resource-based ApplicationSet generator: 
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
 name: guestbook
spec:
 generators:
 - clusterDecisionResource:
    # ConfigMap with GVK information for the duck type resource
    configMapRef: my-configmap      
    name: quak  # Choose either "name" of the resource or "labelSelector"
```

Which might reference an external resource named `quak`, managed by an external controller, containing a list of clusters:
```yaml
apiVersion: mallard.io/v1beta1
kind: Duck
metadata:
  name: quak
spec: {}
status:
  decisions:   # Duck-typing ignores all other aspects of the resource except the "decisions" list
  - clusterName: cluster-01
  - clusterName: cluster-02
```

See the [Cluster Decision Resource](https://argocd-applicationset.readthedocs.io/en/master/Generators-Cluster-Decision-Resource/) generator documentation for more information. [Cluster Decision Resource generator](https://github.com/argoproj-labs/applicationset/pull/231) and [labelSelector support](https://github.com/argoproj-labs/applicationset/pull/272) contributed by [@jnpacker](https://github.com/jnpacker).

### Preserve Application child resources on deletion of parent ApplicationSet

By default, Applications created and managed by the ApplicationSet controller include the [Argo CD resource deletion finalizer](https://argoproj.github.io/argo-cd/user-guide/app_deletion/#about-the-deletion-finalizer). This means that when an ApplicationSet is deleted, its child Applications will be deleted, as well as the cluster resources of those child Applications. This is the same behaviour as a cascade delete within Argo CD.

However, this behaviour is not always desirable: one may want to preserve the Application child resources on deletion of the parent Application. To enable this, using the `.spec.syncPolicy.preserveResourcesOnDeletion` value in the parent ApplicationSet:
```yaml
kind: ApplicationSet
spec:
  generators:
    - clusters: {}
  template:
    # (...)
  syncPolicy:
    # Don't delete Application's child resources, on parent deletion:
    preserveResourcesOnDeletion: true
```

See the [Application Deletion Behaviour](https://argocd-applicationset.readthedocs.io/en/master/Application-Deletion/) documentation for more information. Contributed by [@mmatur](https://github.com/argoproj-labs/applicationset/pull/223).


### Add YAML configuration file support to Git File generator

In the previous ApplicationSet release, only JSON-formatted configuration files were supported by the Git File generator. Now, both [JSON and YAML files are supported](https://argocd-applicationset.readthedocs.io/en/master/Generators-Git/#git-generator-files), and may be used interchangeably. Contributed by [@teejaded](https://github.com/argoproj-labs/applicationset/pull/211).

### Allow any key/value pair in List generator

The List generator previously only supported a fixed list of cluster name/URL values, with an optional values field for user-defined values. You can now specify any key/value pair:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
spec:
 generators:
 - list:
     elements:
     # current form, still supported:
     - cluster: engineering-dev
       url: https://kubernetes.default.svc
       values:
         additional: value
     # new form, does not require cluster/URL keys:
     - staging: true
       gitRepo: https://kubernetes.default.svc   
 template:
   # (...)
```

This new form is fully backwards compatible with existing v0.1.0-originated ApplicationSets, and no migration steps are needed. See the [List generator](https://argocd-applicationset.readthedocs.io/en/master/Generators-List/) documentation for more information.  Contributed by [@ishitasequeira](https://github.com/argoproj-labs/applicationset/pull/290).

### Added additional path params to Git File generator

The Git File generator now produces `{{ path }}` and `{{ path.basename }}` parameters, containing the path of the configuration file, similar to the same parameters already produced by the Git Directory generator. Contributed by [@ishitasequeira](https://github.com/argoproj-labs/applicationset/pull/260).

### Add exclude path support to Git directories

The Git Directory generator scans directories within a Git repository, looking for directories that match specific criteria. However, with the previous ApplicationSet release, there was no way to exclude folders from the scan. 

The ability to [exclude individual paths from being scanned](https://argocd-applicationset.readthedocs.io/en/master/Generators-Git/#exclude-directories) is now supported:
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
      # Include this first path 
      - path: examples/git-generator-directory/excludes/cluster-addons/* 
      # But, exclude this second path:
      - path: examples/git-generator-directory/excludes/cluster-addons/exclude-helm-guestbook
        exclude: true
```        
Contributed by [@LiorLieberman](https://github.com/argoproj-labs/applicationset/pull/195).

### Quality of Life Improvements

- Git Directory generator should ignore all folders starting with '.' ([#201](https://github.com/argoproj-labs/applicationset/pull/201), contributed by @LiorLieberman)
- Add support for custom 'finalizers' in Application templates ([#289](https://github.com/argoproj-labs/applicationset/pull/289), contributed by @tenevdev)
- Enable the ability to set log level on ApplicationSet controller startup ([#249](https://github.com/argoproj-labs/applicationset/pull/249), contributed by @rumstead)
- Add 'appset/appsets' shortnames to the ApplicationSet CRD ([#184](https://github.com/argoproj-labs/applicationset/pull/184), contributed by @christianh814)
- Log appset version during controller startup log ([#241](https://github.com/argoproj-labs/applicationset/pull/241), contributed by @chetan-rns)

### Fixes, infrastructure, and docs

#### Fixes:
- Race condition on cluster creation/deletion causes unexpected behaviour ([#198](https://github.com/argoproj-labs/applicationset/pull/198), contributed by @jgwest)
- Sanitize cluster name param by replacing unsupported character ([#237](https://github.com/argoproj-labs/applicationset/pull/237), contributed by @chetan-rns)
- Remove the unused SkipPrune from ApplicationSetSyncPolicy ([#268](https://github.com/argoproj-labs/applicationset/pull/268), contributed by @ishitasequeira)
- ApplicationSets that initially fail validation might not be re-reconciled ([#296](https://github.com/argoproj-labs/applicationset/pull/296), contributed by @jgwest)
- Remove unnecessary RBAC permissions 'delete events' and 'update events' that are not available at namespace level ([#278](https://github.com/argoproj-labs/applicationset/pull/278), contributed by @vivienfricadelamadeus)
- Issue 170 is not solved when using server, only when using name for destination ([#282](https://github.com/argoproj-labs/applicationset/pull/282), contributed by @jgwest)
- Check path exists passed in application spec ([#253](https://github.com/argoproj-labs/applicationset/pull/253), contributed by @mksha)
- Handle conversion to string in git generator ([#235](https://github.com/argoproj-labs/applicationset/pull/235), contributed by @chetan-rns)
- Add git-lfs to resolve error using files generator ([#215](https://github.com/argoproj-labs/applicationset/pull/215), contributed by @palp)
- Preserve ArgoCD Notification state ([#193](https://github.com/argoproj-labs/applicationset/pull/193), contributed by @dctrwatson)
- Matrix generator's getParams should check error return to avoid panic ([#326](https://github.com/argoproj-labs/applicationset/pull/326), contributed by @jgwest )
- Add missing generators to supported Matrix child generators ([#328](https://github.com/argoproj-labs/applicationset/pull/328), contributed by @jgwest)

#### Test/infrastructure improvements:
- Use a fixed version of Kustomize in the Makefile to generate manifests ([#207](https://github.com/argoproj-labs/applicationset/pull/207), contributed by @jgwest)
- Intermittent E2E test failure on TestSimpleGitFilesPreserveResourcesOnDeletion ([#229](https://github.com/argoproj-labs/applicationset/pull/229), contributed by @jgwest)
- Write additional tests for repo_service.go ([#226](https://github.com/argoproj-labs/applicationset/pull/226), contributed by @jgwest)
- Add more cluster e2e tests ([#148](https://github.com/argoproj-labs/applicationset/pull/148), contributed by @OmerKahani)
- Rebase go.mod to latest Argo CD v2.0, adopt new APIs as needed, and retest ([#281](https://github.com/argoproj-labs/applicationset/pull/281), contributed by @jgwest)
- TestGetFileContent test is failing when running go test, on Mac ([#246](https://github.com/argoproj-labs/applicationset/pull/246), contributed by @jgwest)
- Update build scripts to target 'argoproj/applicationset' on quay.io ([#242](https://github.com/argoproj-labs/applicationset/pull/242), contributed by @jgwest)
- Automatically build and push a container image to argoproj/argocd-applicationset:latest on each master commit ([#256](https://github.com/argoproj-labs/applicationset/pull/256), contributed by @jgwest)
- Split generators into separate doc pages, plus general doc updates ([#267](https://github.com/argoproj-labs/applicationset/pull/267), contributed by @jgwest)
- Use image-push target to build and push an image ([#264](https://github.com/argoproj-labs/applicationset/pull/264), contributed by @chetan-rns)
- Update Argo CD dependency to latest, and update changed APIs ([#310](https://github.com/argoproj-labs/applicationset/pull/310), contributed by @jgwest)
- Increment version for next release ([#188](https://github.com/argoproj-labs/applicationset/pull/188), contributed by @jgwest)
- Use a fixed version of controller-gen in the Makefile to generate manifests ([#233](https://github.com/argoproj-labs/applicationset/pull/233) contributed by @jgwest)
- Update Dockerfile to latest Argo CD base and better cleanup apt dependencies [#302](https://github.com/argoproj-labs/applicationset/pull/302), contributed by @jgwest)
- Add GITHUB_TOKEN and GITLAB_TOKEN to actions and test secrets ([#320](https://github.com/argoproj-labs/applicationset/pull/320))


#### Doc updates:
- Document the policy parameter of the controller ([#297](https://github.com/argoproj-labs/applicationset/pull/297), contributed by @jgwest)
- Update documentation to mention development builds ([#263](https://github.com/argoproj-labs/applicationset/pull/263), contributed by @jgwest)
- Add a note regarding use of set-based requirements ([#228](https://github.com/argoproj-labs/applicationset/pull/228), contributed by @gdubya)
- Refresh README.md based on new documentation ([#191](https://github.com/argoproj-labs/applicationset/pull/191), contributed by @jgwest )
- Fix minor grammar mistake in README ([#217](https://github.com/argoproj-labs/applicationset/pull/217), contributed by @williamcodes)
- Correct list generator syntax ([#204](https://github.com/argoproj-labs/applicationset/pull/204), contributed by @gdubya)
- Update Generators-Git.md 
- Tweak List and Git generator docs ([#333](https://github.com/argoproj-labs/applicationset/pull/333), contributed by @jgwest)
- Generator updates, upgrade instructions, and misc updates ([#313](https://github.com/argoproj-labs/applicationset/pull/313), contributed by @jgwest)

 
## Installation

The ApplicationSet controller must be installed into the same namespace as the Argo CD it is targeting:
```
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/applicationset/v0.2.0/manifests/install.yaml
```

Once installed, the ApplicationSet controller requires no additional setup. You can learn more about ApplicationSet controller installation from the [Getting Started](https://argocd-applicationset.readthedocs.io/en/v0.2.0/Getting-Started/) page.



# v0.1.0

I am excited to announce the first release of the Argo CD ApplicationSet controller, v0.1.0, releasing now alongside Argo CD v2.0!

The ApplicationSet controller provides the ability:
- To deploy Argo CD Applications to multiple Kubernetes clusters at once
- To deploy multiple Argo CD applications from a single monorepo
- Allows unprivileged cluster users (those without access to the Argo CD namespace) to deploy Argo CD applications without the need to involve cluster administrators in enabling the destination clusters/namespaces

BUT, best of all, all these features are controlled by only a single ApplicationSet Kubernetes custom resource, which means no more juggling of multiple Argo CD Application resources to target those multiple clusters/repos!

Unlike with an Argo CD Application resource, which deploys resources from a single Git repository to a single destination cluster/namespace, ApplicationSet uses templated automation to create, modify, and manage multiple Argo CD applications at once.

You can learn more about this from the [ApplicationSet documentation](https://argocd-applicationset.readthedocs.io) or check out the source [and learn how you can contribute](https://github.com/argoproj-labs/applicationset/).

Since this is our first release, we would ❤️ if you would give ApplicationSets a shot, and let us know what cool problems you are solving with it, or what pain points you hit.
Got feature requests, bug reports, or want to contribute code? Let us know on our project repository, or join us on [#argo-cd-appset on Slack](https://argoproj.github.io/community/join-slack).


## Contributors

A great deal of work has gone into bringing this project to life, from many different contributors, all the way from its inception in early 2020 until today. It is terrific to be able to bring all that work together as ApplicationSet's first release, and make it available to a wider audience… we welcome you to try it out, and let us know what you think!
A big thanks to all ApplicationSet controller contributors for their hard work over the last year, whether it be contributing code, writing design documentation, performing code reviews, writing user documentation, and opening issues and PRs:
- Omer Kahani ([@OmerKahani](https://github.com/OmerKahani))
- Devan Goodwin ([@dgoodwin](https://github.com/dgoodwin))
- Michael Goodness ([@mgoodness](https://github.com/mgoodness))
- Alexander Matyushentsev ([@alexmt](https://github.com/alexmt))
- Jonathan West ([@jgwest](https://github.com/jgwest))
- Matteo Ruina ([@maruina](https://github.com/maruina))
- Jesse Suen ([@jessesuen](https://github.com/jessesuen))
- Xianlu Bird ([@xianlubird](https://github.com/xianlubird))
- William Tam ([@wtam2018](https://github.com/wtam2018))
- Ratnadeep Debnath ([@rtnpro](https://github.com/rtnpro))
- John Pitman ([@jopit](https://github.com/jopit))
- Shoubhik Bose ([@sbose78](https://github.com/sbose78))
- Alex Sharov ([@kvendingoldo](https://github.com/kvendingoldo))
- Omer Levi Hevroni ([@omerlh](https://github.com/omerlh))

The ApplicationSet controller would not exist without the contributions of these talented individuals! 🎉️

## New in this Release

### List Generator

The List generator generates parameters based on a fixed list of cluster name/URL values, with those values passed as parameters into the template. This allows manual control over Application destinations via editing of a literal list with the ApplicationSet.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
spec:
 generators:
 - list:
     elements:
     - cluster: engineering-dev
       url: https://kubernetes.default.svc
#    - cluster: engineering-prod
#      url: (another cluster API url)
 template:
   spec:
     project: default
     source:
       repoURL: https://github.com/argoproj-labs/applicationset.git
       targetRevision: HEAD
       path: examples/list-generator/guestbook/{{cluster}}
     # (...)
```       
In this example, if one wanted to add a second cluster, we could uncomment the second cluster element and the ApplicationSet controller would automatically target it with the defined application.

### Cluster Generator

The cluster generator is useful if you are using Argo CD to manage cluster add-ons, such as Custom Resource Definitions (CRDs) or Ingress Controllers, across a fleet of clusters. Instead of manually creating an application for each cluster, you can delegate it to the cluster generator. 

In Argo CD, managed clusters are stored within Secrets in the Argo CD namespace. The ApplicationSet controller uses those same Secrets to generate template parameters for which to target available clusters.

The Cluster generator will automatically identify clusters defined with Argo CD, and extract the cluster data as parameters:
```yaml
kind: ApplicationSet
spec:
  generators:
  - clusters: {} # Automatically use all clusters defined within Argo CD
  template:
    metadata:
      name: '{{name}}-guestbook' # 'name' field of the cluster
    spec:
      source: 
        # (...)
      destination:
        server: '{{server}}' # 'server' field of the cluster
        namespace: guestbook
```


### Git Directory Generator

It is a good practice to define a convention between Argo CD application name and the location of the deployment manifests directory with the Git repository. For example, you might choose to store all manifests of production applications under "applications/prod/<app-name>" and all staging applications under "applications/staging/<app-name>".

The Git Directory generator allows you to go one step further and "codify" that convention. The Git directory generator generates template parameters using the directory structure of a specified Git repository.
Whenever a new subfolder is added to the Git repository, the ApplicationSet controller will detect this change and automatically use the subfolder path to template an Argo CD application containing the manifests contained within that folder (whether they be plain YAML, Kustomize, Helm, etc).

```yaml
kind: ApplicationSet
spec:
  generators:
  - git:
      repoURL: https://github.com/argoproj-labs/applicationset.git
      revision: HEAD
      directories:
      - path: examples/git-generator-directory/cluster-addons/*
# (...)
```

### Git File Generator

Similar to the Directory generator, the Git File generator allows you to generate applications based on Git repository content but provide a bit more flexibility. The Git file generator generates template parameters using the contents of JSON files found within a specified repository.
Git commits containing changes to JSON files are automatically discovered by the Git generator, and the contents of those files are parsed and converted into template parameters.

This allows the creation of custom Argo CD Applications based on the contents of automatically discovered JSON files within the repository. As new files are added/changed, new Argo Applications are created or modified.

```yaml
kind: ApplicationSet
spec:
  generators:
  - git:
      repoURL: https://github.com/argoproj-labs/applicationset.git
      revision: HEAD
      files:
      - path: "examples/git-generator-files-discovery/cluster-config/**/config.json"
```


### Brand new documentation and examples

The ApplicationSet controller has a brand new set of documentation and examples. Topics include introduction, a quick getting started guide, use cases, the interaction with Argo CD, generators, template fields, application lifecycle, and more.

### Template Override

In addition to specifying a template within the `.spec.template` of the ApplicationSet resource, templates may also be specified within generators. This is useful for overriding the values of the spec-level template with generator-specific values.

```yaml
spec:
  generators:
  - list:
      elements:
        - cluster: engineering-dev
          url: https://kubernetes.default.svc
      template: # <--- A template under a list generator
        metadata: {}
        spec:
          project: "default"
          source:
            revision: HEAD
            repoURL: https://github.com/argoproj-labs/applicationset.git
            # New path value is generated here:
            path: 'examples/template-override/{{cluster}}-override'
          destination: {}
```


### Support for arbitrary key/value pairs in Cluster generator and List generator

Arbitrary key/value pairs may be included within the Cluster and List generators, which will be converted into template parameters during template rendering. This is useful for providing custom parameters for a specific generator instance:
```yaml
spec:
  generators:
  - clusters:
      values:
        version: '2.0.0'
```     



### Bugs, tests, and infrastructure improvements

In addition to the above new features, we delivered lots of bug fixes, new unit tests, a new end-to-end test framework, new end-to-end tests, a new release process, and build/test infrastructure improvements.




## Installation

The ApplicationSet controller must be installed into the same namespace as the Argo CD it is targetting:
```
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/applicationset/v0.1.0/manifests/install.yaml
```

Once installed, the ApplicationSet controller requires no additional setup. You can learn more about ApplicationSet controller installation from the [Getting Started](https://argocd-applicationset.readthedocs.io/en/v0.1.0/Geting-Started/) page.



## Changelog

#### Features:
- Add item list CRD Spec (#1)
- Sketch out cluster generator in the CRD, and generator interface (#2)
- Add list generator support (#3)
- Add git directory crd (#4)
- Add git directory generator (#7)
- Cluster Generator reconciling on cluster secret events (#8)
- Create applications (#9)
- feat(cluster-generator): allow templating of app labels (#24)
- feat(cluster-generator): support matchExpressions in selectors (#25)
- Improve controller error handling and logging. (#42)
- Implement Git files discovery (#45)
- add SyncPolicy (#48)
- Add dry run option (#50)
- add requeueAfter option (#51)
- Add template override (#56)
- Add support to metadata.annotations in the template (#58)
- feat(cluster-generator): support arbitrary key:value pairs (#91)
- Add support for support arbitrary key:value pairs in ListGenerator (#110)
- Kustomize integration of Application Set controller and Argo CD (#113)


#### Docs and examples:
- 'User guide' docs: the whats/whys/hows of ApplicationSet controller documentation (#84)
- 'User guide' docs: how to use ApplicationSet controller (examples of generators, template fields) documentation (#85)
- User guide docs: how to use ApplicationSet controller, whats/whys/hows, examples (#117)
- Improve and document the developer deployment process. (#37)
- Expand on existing developer docs: how to setup dev env and how to run ApplicationSet controller on local machine (#74) (#90)
- Making examples more clear (#111)
- Add readthedocs/mkdocs integration to the documentation (#130)
- Fix template-overrides-example.yaml (#144)



#### Tests:
- Add build and test GH action, fix TestGetApps race condition (#83) (#64)
- Implement E2E test framework for testing ApplicationSet against live Kubernetes/Argo CD instance (#65) (#66)
- Intermittent test failure in 'TestGitGenerateParamsFromFiles/handles_error_during_getting_repo_file_contents' (#96)
- Add E2E tests for the git file generator (#138)
- Write tests for pkg/utils/util.go (#146)
- Write tests for clustereventhandler.go (#147)

#### Bugs:
- fix(list-generator): return generated applications (#21)
- Fix rbac error listing Secrets at cluster scope. (#38)
- Fix missing Application RBAC. (#39)
- Log warning if applicationset contains unrecognized generators (#67)
- The --namespace controller param and NAMESPACE environment variable should override to produce one canonical value (#70) (#109)
- Cluster generator cannot select local cluster (#116)
- Fix leader election (#125)
- Git Directory Generator only matches directories that contain valid Helm/ksonnet/Kustomize artifacts (#132)
- Prevent ApplicationSet controller from creating invalid Applications, causing 'unable to delete application resource' in Argo CD (#136)
- Git generator may never detect new commits, if using default 'GetRequeueAfter' value (#137)
- 'Error generating params' error when using JSON array in Git files generator file (#139)
- '`appprojects.argoproj.io` is forbidden' error from serviceaccount argocd-applicationset-controller (#141)
- Applicationset-controller overloads api-server and have memory leak (#153)
- Annotations set in application-set not updating apps (#156)
- Git helpers missing from image (#160)
- ApplicationSet does not support private repos configured using SSH (#163)
- Workaround Argo CD cluster deletion reconciliation bug (#170)

#### Tasks:
- Update applications & add owner reference (#10)
- Refactor the create application method so it will also do update and delete. (#11)
- Reorg manifests & create kustomizations (#20)
- Refactor - Making testing & writing generators more easy (#26)
- Drop cluster-wide RBAC install and fix finalizer permissions issue. (#53)
- Add Apache 2 LICENSE file. (#54)
- Refine ApplicationSet SyncPolicy API to be less like ArgoCD. (#55)
- Fix make deploy step (#60)
- Add go fmt workflow, and go fmt the code (#62)
- Add .golangci.yml from Argo CD, and fix corresponding linter failures (#63)
- Raise error for duplicate application names (#69)
- Sync go.(mod/sum) with argo-cd 1.8 release (#89)
- Re-bootstrap of the project using kubebuilder 2 (#93)
- Add 'lint-go' and 'go mod tidy' to GitHub actions (#95)
- Setup release build scripts/artifacts for ApplicationSet controller releases (#105)
- Kustomize Argo CD integration (and GitHub E2E test action) should use a fixed Argo CD version (#119)
- Use `apiextensions.k8s.io/v1` CustomResourceDefinition, rather than deprecated v1beta1 (#128)
- Update test/generation Argo CD target version to v1.8.5 (#134)
- Adopt same base image as Argo CD to fix vulnerability scan issues (#151)
