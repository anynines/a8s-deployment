# Logging Operational Model

We create a separate namespace for a8s system components such as
the logging framework. This allows us to isolate a8s Kubernetes objects from
the rest of the cluster.

We use a node level approach in the following steps.

Kubernetes logs many things on the node's disk (or uses systemd journal, but
let's ignore it in this document). We need to detect, parse, filter, ... forward
those logs to a destination.

For example the containers are logged in the directory `/var/log/containers/*`
for all the pods that log to `stdout`.

There you can find files in the following style:

```
coredns-74ff55c5b-fjqjc_kube-system_coredns-1ec28490c4597e457fb89873644f25a3989bef9362551e63240d49cdc4de75bd.log
counter_default_count-d171c94305d25168402747e06ec0489f5f11b520388c01796450db3e224337d8.log
etcd-minikube_kube-system_etcd-be2779ad044192e27ee9dbd265504bd23cc9b5f975df0a1db5fdf32805aed7f2.log
```

The filename already provides some valuable information. For example the
`counter` pod is running in namespace `default` and has container id
`d171c94305d25168402747e06ec0489f5f11b520388c01796450db3e224337d8`.

The log files themselves are in `json` file format. They contain the following
fields:
- log
- stream
- time

```
{"log":"[INFO] plugin/reload: Running configuration MD5 = db32ca3650231d74073ff4cf814959a7\n","stream":"stdout","time":"2021-05-27T09:35:09.2441667Z"}
```

In order to now parse those log files, we will use a
[DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/)
, mount relevant directories on the Kubernetes nodes to our pods and use a
software to process the logs within the pods.

# Fluent Bit

## Manual Steps

In the following we will use some Fluent Bit tooling to process the logs on
minikube.

### Creation

First we need to install the Fluent Bit daemonset.
It is configured in a way that it forwards the processed logs to a central
Fluentd instance. Fluent Bit works best as a forwarder given its designed with
performance in mind: high throughput with low CPU and Memory usage. We run it
as a daemonset which ensures that all (or some) Nodes run a copy of a
Pod. As nodes are added to the cluster, Pods are added to them. As nodes are
removed from the cluster, those Pods are garbage collected. This is important
given each Fluent Bit pod can collect and annotate logs for all containers on
its node from a directory on the node.

```bash
kubectl apply -f logging/collection-infrastructure/fluent-bit-node-collector.yaml
```

### Deletion

If you want to get rid of the whole daemon set setup, you can run the following
commands:

```shell
kubectl delete -f logging/collection-infrastructure/fluent-bit-node-collector.yaml
```

# Fluentd

## Manual steps

In the following we will use some Fluentd tooling to process the logs on minikube.

### Creation

First we need to install the Fluentd aggregator.
It is configured in a way that it aggregates the processed logs from Fluent Bit
and then outputs to multiple destinations. Fluentd is well suited to this task
given its wide range of output plugins that provide support for many
destinations that the logs can be sent to. We run it as a StatefulSet so that
uptime can be ensured if pods are spread over multiple availability zones.

We use a custom image which adds some useful plugins to the base Fluentd image.
One of which is the fluent-plugin-label-router so that we can route Fluentd
records based on their Kubernetes metadata. This allows us to specify labels or
some other Kubernetes metadata which can then be used to route specific records
to a different destination. So we have all the `a8s.a9s/dsi-name:sample-pg-cluster`
labeled pods (an instance of a PostgreSQL cluster) directed to stdout, this
could be any destination. Additionally, we used the copy input plugin to
duplicate the records so that we could have all logs in the cluster sent to a
global destination, like OpenSearch, instead of simply routing the
PostgreSQL instance traffic to a single destination.

```shell
cd Images/fluentd-aggregator/
export IMG=localhost:5000/fluentd
docker build -t $IMG .
docker push $IMG
cd ../..
```

```shell
kubectl apply -f logging/collection-infrastructure/fluentd-aggregator.yaml
```

We will also deploy a PostgreSQL cluster and OpenSearch so that we can see logs
in a real cluster being sent to multiple destinations by the Fluentd aggregator.
Follow the instructions from the [a8s-demo][a8s-demo] in order to get a cluster
up and running using our PostgreSQL Operator.

Once this has been applied, you should see some log messages that pertain to
the PostgreSQL cluster in logs of the fluentd-aggregator in JSON format. You
will also see logstash format logs which are being logged before being sent to
OpenSearch. You can differentiate between these based on the
different formats.

```shell
kubectl -n a8s-system logs a8s-fluentd-aggregator-0
```

### Deletion

If you want to get rid of the whole daemon set setup, you can run the following
commands:

```shell
kubectl delete -f logging/collection-infrastructure/fluentd-aggregator.yaml
```

[a8s-demo]: https://github.com/anynines/a8s-demo
