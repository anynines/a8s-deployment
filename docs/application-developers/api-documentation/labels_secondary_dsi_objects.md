# Labels of Secondary Data Service Instances Resources

The labels of secondary resources belonging to a DSI, such
as pods, are the union of three groups of labels :

```yml
labels:
    <a8s reserved labels>
    <DSI labels>
    <DataService specific labels>
```

`<a8s reserved labels>` are labels that a8s manages and needs to correctly
manage the DSIs. These labels are shown below:

```yml
labels:
    a8s.a9s/dsi-group: <API group of the DSI CRD>   
    a8s.a9s/dsi-kind: <kind of the CR>
    a8s.a9s/dsi-name: <name of the instance defined in the CR>
```

`<DSI labels>` are just the metadata labels of the DSI CustomResource.

Some data services may have additional labels, you will find them in the
following sections.

## PostgreSQL

The PostgreSQL pods also have the following labels:

```yml
  a8s.a9s/replication-role: <role of the pod, either master or replica>
  # Managed by Kubernetes
  controller-revision-hash: <contains hash of current pod template>
  statefulset.kubernetes.io/pod-name: <name of the pod in the statefulset>
```

For example the master pod of a PosgreSQL instance with labels 

```yml
labels:
  test-label: "test"
```

will have the labels:

```yml
labels:
  a8s.a9s/dsi-group: postgresql.anynines.com
  a8s.a9s/dsi-kind: Postgresql
  a8s.a9s/dsi-name: sample-pg-cluster
  a8s.a9s/replication-role: master
  controller-revision-hash: sample-pg-cluster-86765dfc5d
  statefulset.kubernetes.io/pod-name: sample-pg-cluster-0
  test-label: test
```
