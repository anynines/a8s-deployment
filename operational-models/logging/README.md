# Logging Operational Model

Based on minikube.

## Manual steps

We create a separate namespace for a8s system components such as
the logging framework. This allows us to isolate a8s Kubernetes objects from
the rest of the cluster.

```shell
kubectl apply -f dashboard/a8s-system.yaml
```

We use a node level approach in the following steps.

Kubernetes logs many things on the node's disk (or uses systemd journal, but
let's ignore it in this document). We need to detect, parse, filter, ... forward
those logs to a destination.

For example the the containers are logged in the directory
`/var/log/containers/*` for all the pods that log to `stdout`.

There you can find files in the following style:

```
coredns-74ff55c5b-fjqjc_kube-system_coredns-1ec28490c4597e457fb89873644f25a3989bef9362551e63240d49cdc4de75bd.log
counter_default_count-d171c94305d25168402747e06ec0489f5f11b520388c01796450db3e224337d8.log
etcd-minikube_kube-system_etcd-be2779ad044192e27ee9dbd265504bd23cc9b5f975df0a1db5fdf32805aed7f2.log
```

The filename already provides some valuable information. For example the
`counter` pod is running in namespace `default` and has container id
`d171c94305d25168402747e06ec0489f5f11b520388c01796450db3e224337d8`.

The log files themself are in `json` file format. They contain the following
fields:
- log
- stream
- time

```
{"log":"[INFO] plugin/reload: Running configuration MD5 = db32ca3650231d74073ff4cf814959a7\n","stream":"stdout","time":"2021-05-27T09:35:09.2441667Z"}
```

In order to now parse those log files, we will use a
[DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/)
, mount directories on the Kubernetes nodes to our pods and use a software to
process the logs within the pods.

Some people already built tools to process those kind of logs, so we can base
our work on that. There are for example [Fluentd](https://www.fluentd.org/) and
[Fluent Bit](https://fluentbit.io/) available.

In the following we will use some Fluentd tooling to process the logs on minikube.


### Creation

First we need to install the fluentd daemonset.
It is configured in a way that it forwards the processed logs to a syslog
destionation. So you might want to change the destination host/ip.

```bash
kubectl apply -f logging/fluentd-daemonset-permissions.yaml
kubectl apply -f logging/fluentd-daemonset-syslog-minikube.yaml
```

Once this has been applied, you should see some log message at your syslog
destination.

You can use a demo app to see some application specific logs at the syslog
destination:

```shell
kubectl apply -f logging/demo-app-counter.yaml
```

### Deletion

If you want to get rid of the whole daemon set setup, you can run the following
commands:

```shell
kubectl delete -f logging/fluentd-daemonset-minikube.yaml
kubectl delete -f logging/fluentd-daemonset-permissions.yaml
```

If you used the demo app, delete it using the following commands:

```shell
kubectl delete -f logging/demo-app-counter.yaml
```

### Notes
