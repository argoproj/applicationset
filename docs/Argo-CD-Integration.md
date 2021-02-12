# How ApplicationSet controller interacts with Argo CD

When you create, update, or delete an `ApplicationSet` resource, the ApplicationSet controller responds by creating, updating, or deleting, one or more corresponding Argo CD `Application` resources.

In fact, the sole responsibility of the ApplicationSet controller is to create, update, and delete `Application` resources within the target namespace. It ensures that the `Application` resources remain consistent with the defined declarative `ApplicationSet` resource, and nothing more.

It is Argo CD itself that is responsible for the actual deployment of the generated child `Application` resources (eg Deployments, Services, ConfigMaps, etc). The ApplicationSet controller is *only* responsible for the contents of the `Application` resource.

Thus the ApplicationSet controller:
- Does not create/modify/delete Kubernetes resources (other than the `Application` CR)
- Does not connect to clusters other than the one Argo CD is deployed to
- Does not interact with namespaces other than the one Argo CD is deployed within

The ApplicationSet controller can thus be thought of as an `Application` 'factory', taking an `ApplicationSet` as input, and generating as output one or more Argo CD `Application` resources that correspond to the parameters of that set.

![ApplicationSet controller vs Argo CD, interaction diagram](assets/Argo-CD-Integration/ApplicationSet-Argo-Relationship-v2.png)

In this diagram, an `ApplicationSet` resource exists, and it is the ApplicationSet controller that is responsible for creating the corresponding `Application` resources. 

The resulting `Application` resources are then managed Argo CD: that is, Argo is responsible for actually deploying the child resources. 

Argo CD generates the application's Kubernetes resources based on to the contents of the Git repository defined within the Application `spec`, deploying e.g. Deployments, Service, and other resources.

Creation, updates, or deletions of ApplicationSets will have a direct effect on the Applications present in the Argo CD namespace. Likewise, cluster events (the addition/deletion of Argo CD cluster secrets, when using Cluster generator), or changes in Git (when using Git generator), will be used as input to the ApplicationSet controller in constructing `Application` resources.

Argo CD and the ApplicationSet controller work together to ensure a consistent set of Application resources exist, and are deployed across the target clusters.
