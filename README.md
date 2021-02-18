# Argo CD ApplicationSet CRD

The Argo CD ApplicationSet CRD and controller provides a formal solution to replace the
[app-of-apps](https://argoproj.github.io/argo-cd/operator-manual/cluster-bootstrapping/) pattern
with the ultimate goal of introducing ApplicationSet as a first class supported object in 
Argo CD Core.

This project will initially be maintained separately from Argo CD, in order to allow quick iteration
of the spec and implementation, without tying it to Argo CD releases. No promises of backwards
compatibility are made, at least until merging into Argo CD proper.

## Example Spec:

```yaml
# This is an example of a typical ApplicationSet which uses the cluster generator.
# An ApplicationSet is comprised with two stanzas:
#  - spec.generator - producer of a list of values supplied as arguments to an app template
#  - spec.template - an application template, which has been parameterized
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: guestbook
spec:
  generators:
  - clusters: {}
  template:
    metadata:
      name: '{{name}}-guestbook'
    spec:
      source:
        repoURL: https://github.com/infra-team/cluster-deployments.git
        targetRevision: HEAD
        chart: guestbook
      destination:
        server: '{{server}}'
        namespace: guestbook
```

See the documentation for an explanation of fields and additional examples.

## Documentation

Read [the documentation](https://argocd-applicationset.readthedocs.io/en/stable/) for more information on how to setup and run the ApplicationSet controller, and to learn more about features and usage.

The original design doc is available here: https://docs.google.com/document/d/1juWGr20FQaJmuuTIS8mBFmWWDU422M_FQMuhp5c1jt4/edit?usp=sharing

## Development

Learn more in the documentation on how to [setup a development environment, and build the controller](https://argocd-applicationset.readthedocs.io/en/stable/).