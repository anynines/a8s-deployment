# Platform Operators Documentation

This file contains platform operator specific documentation on how the Kubernetes cluster needs
to be configured.

- [General Prerequisites](#general-prerequisites)
- [Install the a8s Control Plane ](#install-the-a8s-control-plane-for-platform-operators)
  - [Prerequisites](#prerequisites)
  - [Install the a8s Control Plane](#install-the-a8s-control-plane)
  - [(Optional) Install the Logging Infrastructure](#optional-install-the-logging-infrastructure)
  - [Uninstall the Logging Infrastructure](#uninstall-the-logging-infrastructure)
  - [(Optional) Install the Metrics Infrastructure](#optional-install-the-metrics-infrastructure)
  - [Uninstall the Metrics Infrastructure](#uninstall-the-metrics-infrastructure)

## General Prerequisites

You'll need:

- a running Kubernetes cluster
- `kubectl` v1.14 or higher pointed to the Kubernetes cluster
- one [StorageClass][storage-class] marked as `default` in the Kubernetes cluster

The instructions in this repo have been tested on `minikube v1.17.1`, `minikube v1.21.0` and
Kubernetes `v1.20.2`, but they should work with any recent Kubernetes version. Please let us
know if you encounter any issue with other versions.

## Install the a8s Control Plane

### Prerequisites

a8s supports taking backups of data service instances (DSIs). Currently, the backups are stored in
an AWS S3 bucket, so before installing a8s **you must create an AWS S3 bucket that a8s will use to
store backups** ([here][s3-bucket-creation] is the official S3 documentation).

Then, create an access key and secret key for the bucket. This is the key the a8s control plane
will use to interact with the bucket.

After you've created the access key and secret key, you must place the information about the S3
bucket in some files as shown in the following commands. When you'll execute the commands to install
a8s, the content of such files will be used to populate configmaps and secrets that the a8s control
plane will read to be able to upload and download backups from S3. You MUST use the file names shown
in the commands.

```shell
echo <bucket-access-key-id> > deploy/a8s/access-key-id # create file that stores the ID of the key

echo <bucket-secret-access-key > deploy/a8s/secret-access-key # create file that stores the secret value of the key

cp deploy/a8s/backup-store-config.yaml.template deploy/a8s/backup-store-config.yaml # create file with other information about the bucket
```

Then, use an editor to open `backup-store-config.yaml` and replace the value:

- of the `container` field with the name of the S3 bucket
- of the `region` field with the name of the region where the bucket is located

All the created files are gitignored so you don't have to worry about committing them by mistake
(since they contain private data).

### Install the a8s Control Plane

Just run:

```shell
kubectl apply --kustomize deploy/a8s/
```

This command will create the Kubernetes resources that make up the a8s control plane in the correct
order.

More precisely, it will:

1. Create a namespace...

It might take some time for the a8s control plane to get up and running. To know when that happens
you can run the two following commands and wait until both of them show that all deployments
are ready (value `1/1` under the `READY` column):

```shell
watch kubectl get deployment --namespace a8s-system
watch kubectl get deployment --namespace postgresql-system
```

the output of the first command should be similar to:

```shell
NAME                                 READY   UP-TO-DATE   AVAILABLE   AGE
a8s-backup-controller-manager        1/1     1            1           105s
service-binding-controller-manager   1/1     1            1           105s
```

the output of the second command should be similar to:

```shell
NAME                            READY   UP-TO-DATE   AVAILABLE   AGE
postgresql-controller-manager   1/1     1            1           25m
```

### (Optional) Install the Logging Infrastructure

This repo also comes with yaml manifests that you can use to optionally install components to
collect and visualize logs of the provisioned data service instances.

More precisely, these are:

1. A Fluent Bit daemonset where each node-local daemon collects the logs of the Pods on its node.
2. A FluentD aggregator that collects and aggregates the logs from the Fluent Bit daemonset.
3. Elasticsearch and Kibana to query and visualize the logs.

To install them, simply run:

```shell
kubectl apply --recursive --filename deploy/logging/
```

To wait for all components to be up and running, run:

```shell
watch kubectl get pod --namespace a8s-system --selector=a8s.anynines/logging
```

and wait until you see that all the pods (5 + the number of worker nodes of your Kubernetes cluster)
are running (value `1/1` under the `READY` column):

```shell
NAME                                        READY   STATUS    RESTARTS   AGE
a8s-fluentd-aggregator-0                    1/1     Running   0          5m3s
a8s-opendistro-es-client-67f8754767-f4vrl   1/1     Running   1          5m3s
a8s-opendistro-es-data-0                    1/1     Running   1          5m2s
a8s-opendistro-es-kibana-8564d67997-m4z6q   1/1     Running   0          5m1s
a8s-opendistro-es-master-0                  1/1     Running   1          5m2s
fluent-bit-777zx                            1/1     Running   0          5m3s
```

### Uninstall the Logging Infrastructure

Run:

```shell
kubectl delete --recursive --filename deploy/logging/
```

### (Optional) Install the Metrics Infrastructure

Just as for logging this repo includes yaml manifests that can be used to optionally install
components to collect and visualize metrics of the provisioned data service instances. As of
now we do not isolate tenants which means that everyone with access to the Prometheus instance
can see both the system metrics of the Kubernetes control plane as well all metrics that are
scraped from data service instances.

These include:

1. A cluster-level Prometheus deployment that scrapes both the Kubernetes system metrics
   as well as the data service instances.
2. A Grafana dashboard to query and visualize the metrics.

To install them, simply run:

```shell
kubectl apply --recursive --filename deploy/metrics/
```

To wait for all components to be up and running, run:

```shell
watch kubectl get pod --namespace a8s-system --selector=a8s.anynines/metrics
```

and wait until you see the that the Prometheus and Grafana Pods are running
(value `1/1` under the `READY` column):

```shell
NAME                                        READY   STATUS    RESTARTS   AGE
a8s-system    pod/grafana-64c89f57f7-v7wlp                1/1     Running   0          111s
a8s-system    pod/prometheus-deployment-87cc8fb88-225v4   1/1     Running   0          111s
```

### Uninstall the Metrics Infrastructure

Run:

```shell
kubectl delete --recursive --filename deploy/metrics/
```

[storage-class]: https://kubernetes.io/docs/concepts/storage/storage-classes/
[s3-bucket-creation]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html
[mount-secret-in-env-vars]: https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables
[mount-secret-in-volume]: https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod
[kubernetes-ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/
[kubernetes-port-forwarding]: https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/
