# List Generator

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
