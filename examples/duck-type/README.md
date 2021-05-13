# How the duck type generator works
1. The Duck type generator reads the following status format:
```yaml
status:
  decisions:
  - clusterName: cluster-01
  - clusterName: cluster-02
```
2. Any resource containing this list of clusterNames can be referenced by the ApplicationSet Duck Type Generator.
3. The names must match the cluster names define in Argo CD
4. The Service Account used by the ApplicationSet controller must have access to `Get` the resource you want to retrieve the duck type definition from
5. Any cluster name in the `Status.Decisions` list will be matched to an Argo CD known cluster and then an application will be created from the ApplicationSet template

# Applying the example
1. Connect to a cluster with the ApplicationSet controller running
2. Edit the Role for the ApplicationSet service account, and grant it permission to `get` the `placementdecisions` resources, from apiGroups `cluster.open-cluster-management.io/v1alpha1`
```yaml
- apiGroups:
  - "cluster.open-cluster-management.io/v1alpha1"
  resources:
  - placementdecisions
  verbs:
  - get
```
3. Apply the following CRD to allow creating of placementdecision custom resources:
```bash
kubectl apply -f https://raw.githubusercontent.com/open-cluster-management/api/main/cluster/v1alpha1/0000_04_clusters.open-cluster-management.io_placementdecisions.crd.yaml
```
4. Now apply the PlacementDecision and an ApplicationSet:
```bash
kubectl apply -f ./placementdecision.yaml
kubectl apply -f ./ducktype-example.yaml
```
5. For now this won't do anything until you create a controller that populates the `Status.Decisions` array.