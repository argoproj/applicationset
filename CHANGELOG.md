# Changelog

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

The ApplicationSet controller would not exist without the contributions of these talented individuals! 🎉

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
