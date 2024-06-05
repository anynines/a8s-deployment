# Advanced Configuration of a Data Service Instance

This file will guide you through setting up more advanced configuration options
of a PostgreSQL instance.

## Index

- [Usage Outside the Kubernetes Cluster](#usage-outside-the-kubernetes-cluster)
- [High Availability & Scheduling Constraints](#high-availability--scheduling-constraints)
    - [Affinity and Anti-Affinity](#affinity-and-anti-affinity)
        - [Example : High Availability 1 - Distributing Replicas to Zones](#example--high-availability-1---distributing-replicas-to-zones)
        - [Example: High Availability 2 - More Replicas than Zones](#example-high-availability-2---more-replicas-than-zones)
    - [Taints and Tolerations](#taints-and-tolerations)
        - [Example: Node dedicated to PostgreSQL Instance](#example-node-dedicated-to-postgresql-instance)
    - [Caveats and Known Limitations](#caveats-and-known-limitations)

## Usage Outside the Kubernetes Cluster

To use your instances outside the K8s cluster they are running in, you can expose them via a load
balancer. You can do so by setting the field `spec.expose` equal to `LoadBalancer`, as in this
example:

```yaml
apiVersion: postgresql.anynines.com/v1beta3
kind: Postgresql
metadata:
  name: sample-pg-cluster
spec:
  version: 14
  expose: LoadBalancer
```

The operator will set up a K8s ConfigMap containing the hosts and ports you can use to connect to
your instance.

The name of the ConfigMap will be `<instance-name>-connection`, the namespace will be the same as
the instance's. You can find it and inspect the contents by running:

```sh
kubectl get configmap <instance-name>-connection -o yaml
```

*Note 1*: Exposing the instance will always create at least one dedicated load balancer, which might
cause additional cost, depending on your infrastructure.

*Note 2*: If your instance also has a read-only service (`spec.enableReadOnlyService: true`),
exposing it outside the cluster will create two load balancers, one for the read-only service and
one for the read-write service. This might cost even more (the actual amount depends on your
infrastructure).

*Note 3*: Some infrastructure providers might not support load balancers, if you run a8s on one such
provider even if you specify `spec.expose: LoadBalancer` the instance won't get a load balancer.

## High Availability & Scheduling Constraints

With the help of scheduling constraints you can make better use of your clusters
resource and/or make PostgreSQL instances more resilient against failures by
setting up highly available instances. In general, these settings are exposed
through the `spec.schedulingConstraints` field for example in the `Postgresql`
objects (see [API
Documentation](/docs/application-developers/api-documentation/postgresql-operator/v1beta3.md#postgresqlschedulingconstraints)).

Subfields of `schedulingConstraints` allow you to configure
[tolerations][taints-and-tolerations], [node
affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity),
[pod
(anti-)affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#inter-pod-affinity-and-anti-affinity)
for the Pods of the Data
Service Instances (DSI). They are directly copied, unmodified, to the
corresponding fields of the DSI pods.

Thus the a8s framework fully relies on Kubernetes mechanisms and inherits its
limitations, therefore it is highly recommended going through the Kubernetes
documentation on the topic:
- [Affinity and Anti-Affinity][affinity]
- [Taints and Tolerations][taints-and-tolerations]


The next sections will guide you through the configuration process.

> Note:
> Be careful when assigning scheduling constraints, this can lead to Pods never
> being scheduled and when modifying nodes (for example with taints) it is
> possible to evict all running pods !

### Affinity and Anti-Affinity

[Affinity and Anti-Affinity][affinity] is used to attract or repel pods to/from
K8s cluster nodes at scheduling time or runtime, based on the nodes labels or on
the labels of other pods running on the nodes. The former case is called node
affinity and the latter one inter-pod (anti-)affinity.

You can read more about what constraints are possible in the [Kubernetes
documentation][affinity], as mentioned before the a8s framework does not place
any restrictions on what constraints you can apply.

We will demonstrate how to specify anti-affinity in two simple examples here,
affinity can be expressed analogously.

#### Example : High Availability 1 - Distributing Replicas to Zones

In this section we will assume that:

- you are using a cluster that has nodes in at least 3 availability zones (AZ).
- you want to use a 3 replica PostgreSQL instance.
- the AZ of a node is indicated by the node's label
  `topology.kubernetes.io/zone`.

In this case, for a high available PostgreSQL, the replicas have to be distributed
among those AZs, here is an example `Postgresql` CustomResource
(CR) object that will achieve this:

```yml
apiVersion: postgresql.anynines.com/v1beta3
kind: Postgresql
metadata:
  name: ha-1-sample-pg-cluster
spec:
  replicas: 3
  version: 14
  resources:
    requests:
      cpu: 100m
    limits:
      memory: 200Mi
  schedulingConstraints:
    affinity:
      podAntiAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
            labelSelector:
                matchExpressions:
                - key: a8s.a9s/dsi-name
                  operator: In
                  values:
                  - ha-1-sample-pg-cluster
                - key: a8s.a9s/dsi-kind
                  operator: In
                  values:
                  - Postgresql
            topologyKey: topology.kubernetes.io/zone
```

Let's go through the specs in detail:

```yml
podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
```

Since we want PostgreSQL pods to repel each other, we use **anti-affinity**
here, the `requiredDuringScheduling` part will then indicate which conditions **must** be met
before a pod gets scheduled.

The `IgnoredDuringExecution` implies that in case we are modifying an already
running instance, pods will not get evicted to enforce this policy, so it will
only take effect when pods restart. Therefore, if you want to try out the
examples make sure to always create a new instance.

Goal of our constraints is to express, that no other pod of the same DSI should
be in the same zone, which is done through:

```yml
- podAffinityTerm:
  labelSelector:
    matchExpressions:
    - key: a8s.a9s/dsi-name
      operator: In
      values:
      - ha-1-sample-pg-cluster
    - key: a8s.a9s/dsi-kind
      operator: In
      values:
      - Postgresql
  topologyKey: topology.kubernetes.io/zone
```

Here we only use one `podAffinityTerm` (multiple ones are possible), which
matches pods with the value `ha-1-sample-pg-cluster` in the label
`a8s.a9s/dsi-name` and with value `Postgresql` in the label `a8s.a9s/dsi-kind`.
You can find out how pods are labeled in the [reference
section](/docs/application-developers/api-documentation/labels_secondary_dsi_objects.md).
Since we are specifying an anti-affinity term, a Pod won't be scheduled on a
node in an AZ in which there are already pods that match the match expressions
(which are ANDED). The constraint works at the AZ level, because we used
`topology.kubernetes.io/zone` as `topologyKey`. Whereas `kubernetes.io/hostname`
for example would enforce that there are no other matching pods running on the
same node rather than AZ.

For other possible values see the [well known labels and annotations
section][well-known-annotations].

> Note:
> Although it is best practice to use the well known [labels and
> annotations][well-known-annotations] such as `topology.kubernetes.io/zone` and
> most providers use them, Kubernetes does not enforce them. Thus, you will
> have to make sure that your cluster uses them or replace them with the ones
> that are used in your cluster.
> You can find out by asking your admin or inspecting your cluster nodes.

You can test this example using:

```bash
kubectl apply -f examples/postgresl-ha-1-instance.yaml
```

Then get the nodes where the DSI individual replicas are running:

```bash
kubectl get pods -l a8s.a9s/dsi-name=ha-1-sample-pg-cluster  -o go-template='{{range .items}}{{printf "%s : %s\n" .metadata.name .spec.nodeName }}{{end}}'
```

And verify that the nodes are part of different AZs by inspecting the output of:

```bash
kubectl get nodes -o go-template='{{range .items}}{{printf "%s : %s\n" .metadata.name  (index .metadata.labels "topology.kubernetes.io/zone") }}{{end}}'
```

> Note:
> The documentation warns you that specifying pod constraints can result in
> significantly increased amount of processing at scheduling time, which can
> slow down your cluster.

For a more detailed and complete guide, please refer to the [Kubernetes
documentation][affinity], everything mentioned there is directly applicable to
the a8s framework.

#### Example: High Availability 2 - More Replicas than Zones

While the above example was easy to set up, in a production grade cluster you
might want to have more DSI replicas than AZ, especially to ensure upscaling is
possible without having to worry about scheduling fields.

We are now going to assume you have a cluster with 3 AZs which all contain
multiple nodes and want to run a 5 replicas DSI on it. We want to ensure that
pods are distributed among zones and also that there are no two Pods of the same
DSI running on the same node. This can be achieved using:

```yml
apiVersion: postgresql.anynines.com/v1beta3
kind: Postgresql
metadata:
  name: ha-2-sample-pg-cluster
spec:
  replicas: 5
  version: 14
  resources:
    requests:
      cpu: 100m
    limits:
      memory: 200Mi
  schedulingConstraints:
    affinity:
      podAntiAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
            labelSelector:
                matchExpressions:
                - key: a8s.a9s/dsi-name
                  operator: In
                  values:
                  - ha-2-sample-pg-cluster
                - key: a8s.a9s/dsi-kind
                  operator: In
                  values:
                  - Postgresql
            topologyKey: kubernetes.io/hostname
        preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 10
            podAffinityTerm:
              labelSelector:
                  matchExpressions:
                  - key: a8s.a9s/dsi-name
                    operator: In
                    values:
                    - ha-2-sample-pg-cluster
                  - key: a8s.a9s/dsi-kind
                    operator: In
                    values:
                    - Postgresql
              topologyKey: topology.kubernetes.io/zone
```

Here we chose 5 replica DSI (instead of 3 as in Example 1). If we used the same
schedulingConstraints as the previous example, 2 of the pods would not be
scheduled (we are still assuming 3 AZs), since the `required` constraint
wouldn't be satisfiable. Instead, we have now made this constraint
**preferable**:

```yml
- weight: 10
  podAffinityTerm:
    labelSelector:
      matchExpressions:
        - key: a8s.a9s/dsi-name
          operator: In
          values:
          - ha-2-sample-pg-cluster
        - key: a8s.a9s/dsi-kind
          operator: In
          values:
          - Postgresql
      topologyKey: topology.kubernetes.io/zone
```

This will not prevent scheduling of pods in the same availability zone, but will
minimize the likelihood of having them in the same AZ.

Additionally, when using `preferredDuringSchedulingIgnoredDuringExecution` one
has to give each constraint a weight. This weight conveys to the scheduler how
important the constraint is with respect to other constraints. This is needed
because you can specify multiple constraints and not all of them might be
satisfiable at the same time.

To prevent scheduling on the same node, we modified the `required` constraint to
use the aforementioned `topologyKey` for nodes, i.e. `kubernetes.io/hostname`.

Now scaling up is only limited by the number of Kubernetes nodes. So in this case
you would need 5 nodes, otherwise our `requiredDuringScheduling` constraint will
again prevent 2 pods from scheduling.
If you want to avoid that, you can move the constraint from `required` to
`preferred` and for example give it a weight of 100.

You can apply the example by using:

```bash
kubectl apply -f examples/postgresql-ha-2-instance.yaml
```

Then get the nodes where the DSI individual replicas are running to verify that
they are all different:

```bash
kubectl get pods -l a8s.a9s/dsi-name=ha-1-sample-pg-cluster  -o go-template='{{range .items}}{{printf "%s : %s\n" .metadata.name .spec.nodeName }}{{end}}'
```

And additionally verify that the nodes are not running in a single AZs by
inspecting the output of:

```bash
kubectl get nodes -o go-template='{{range .items}}{{printf "%s : %s\n" .metadata.name  (index .metadata.labels "topology.kubernetes.io/zone") }}{{end}}'
```

### Taints and Tolerations

Where affinity and anti-affinity are used to specify preferences of pods to be
scheduled on specific nodes, [taints and tolerations][taints-and-tolerations]
can be used to prevent pods from being scheduled on a specific node.

A node will be tainted with a certain taint, causing only pods with a matching
toleration to be scheduled/executed on that node. Other pods, in case
of a `NoSchedule` taint will not be scheduled on the node anymore or will even
be evicted from it in case of a `NoExecute` taint. For example, a commonly used
taint is `node-role.kubernetes.io/control-plane` which is typically used on the
nodes reserved for the Kubernetes control plane components.

Since taints can stop pods, be careful when tainting your nodes, you could end
up leaving the cluster in a broken state. Please read the
[documentation][taints-and-tolerations] before using taints and tolerations.

#### Example: Node dedicated to PostgreSQL Instance

First we have to apply a taint to a node, here we will assign the taint
`pg-node`:

```bash
kubectl taint nodes <node_name> pg-node=true:NoSchedule
```

Now to additionally be able to express affinity to that node, you will also have
to label the node using:

```bash
kubectl label nodes <node_name> pg-node=true
```

After that we can specify a PostgreSQL instance with pods able to schedule on
that node and which are attracted to that node:

```yml
apiVersion: postgresql.anynines.com/v1beta3
kind: Postgresql
metadata:
  name: toleration-sample-pg-cluster
spec:
  replicas: 3
  version: 14
  resources:
    requests:
      cpu: 100m
    limits:
      memory: 200Mi
  schedulingConstraints:
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
          - matchExpressions:
            - key: pg-node
              operator: In
              values:
              - "true"
    tolerations:
    - key: "pg-node"
      operator: "Equal"
      value: "true"
      effect: NoSchedule
```

The toleration for our taint is specified in

```yml
tolerations:
  - key: "pg-node"
    operator: "Equal"
    value: "true"
    effect: NoSchedule
```

and by using the `nodeAffinity` term:

```yml
nodeAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
    nodeSelectorTerms:
    - matchExpressions:
      - key: pg-node
        operator: In
        values:
        - "true"
```

We ensure that the pods can only schedule on our tainted and labeled node.

You can apply an instance with that spec by using:

```bash
kubectl apply -f examples/postgresql-toleration-instance.yaml
```

If you now add another DSI, for example our `sample-pg-cluster` from the usage
overview, by using:

```bash
kubectl apply -f examples/postgresql-instance.yaml
```

You will see that none of its replicas will be scheduled on the tainted node.

### Caveats and Known Limitations

This section will point out **some** of the current limitations and caveats you
might be experiencing when working with scheduling constraints, the indicated
Kubernetes documentation pages will provide a more complete overview.

- Using scheduling constraints can evict pods and cause some pods to not be
  scheduled regardless of resources available on the cluster. This is especially
  true when tainting, all pods will either be evicted from a tainted node or
  will not be able to reschedule there. Also, adding `requiredDuringScheduling`
  constraints can prevent scheduling, so be especially careful when using them.
- Kubernetes uses some well known taints defined
  [here](https://kubernetes.io/docs/reference/labels-annotations-taints/), those
  should not be used outside their described use case. Otherwise, other
  workloads or constraints that depend on them might show an unexpected behavior.
- For DSIs that are backed by a StatefulSet (e.g. PostgreSQL), updating
  scheduling constraints from a value that prevents scheduling to a satisfiable
  value won't have an effect. This is due to the StatefulSet controller not
  being able to update the schedulingConstraints while waiting for pods to
  schedule (see [Issue](https://github.com/kubernetes/kubernetes/issues/67250)).
  In this case delete the instance first, before reapplying the valid
  manifest.
- The Kubernetes scheduler does not only take into account your specified
  constraints, but also for example resources available on a node. This can in
  some cases, overrule some of your `prefered` affinity constraints or cause
  some pods with required constraints to be stuck in pending without being
  scheduled for a long time.
- Specifying scheduling constraints and in particular `podAffinity` will
  increase the processing needs of the scheduler, possibly slowing down your
  cluster.
- For some storage classes, the PersistentVolumeClaims can cause the pod to
  stick to a specific node. For example, if a pod was already scheduled on the
  node and after a change in scheduling constraints it can no longer run on it
  the pod can get stuck in pending. The reason is that the PersistentVolumeClaim
  of the pod  is bound to the node and therefore this node becomes the only
  eligible node for scheduling the pod, but the constraints forbid scheduling.
  This behavior will be addressed in future releases of Kubernetes.

[affinity]:https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity
[taints-and-tolerations]:https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
[well-known-annotations]:https://kubernetes.io/docs/reference/labels-annotations-taints
