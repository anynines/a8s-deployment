## Install the a8s Control Plane
 - [Prerequisites](#prerequisites)
 - [RBAC](#rbac)
 - [Install the a8s Control Plane](#install-the-a8s-control-plane-1)
 - [(Optional) Install the Logging Infrastructure](#optional-install-the-logging-infrastructure)
 - [Uninstall the Logging Infrastructure](#uninstall-the-logging-infrastructure)
 - [(Optional) Install the Metrics Infrastructure](#optional-install-the-metrics-infrastructure)
 - [Uninstall the Metrics Infrastructure](#uninstall-the-metrics-infrastructure)

### Prerequisites

#### Installing the Cert-Manager

The a8s framework relies on the [certificate manager][cert-manager] to generate
TLS certificates, therefore you will first have to install it on your cluster.

> Note that when following this guide make sure to check the [cloud
> compatibility page][cert-manager-compatibility] on configuration needs of your
> cluster. 

In general there are a multitude of [installation
options](https://cert-manager.io/docs/installation/) supported by the
cert-manager, for this guide we will only describe how to setup a basic
deployment, since that suffices for the framework. For a production grade
deployment please consult the [documentation][cert-manager].

To setup a basic deployment, use:
```bash
kubectl apply -k deploy/cert-manager
``` 

This will install all necessary CRDs and additionally create a simple deployment
of the cert-manager components. After that you are done with the setup for the
a8s framework, since we currently only use self signed certificates. 

If you want to use certificates for your components in production, you might
want to check out the [configuration pages][cert-manager-config], where you can
for example find instructions on how to setup a CA.

#### Configuring Backups

a8s supports taking backups of data service instances (DSIs). Currently, the backups are stored in
an AWS S3 bucket, so before installing a8s **you must create an AWS S3 bucket that a8s will use to
store backups** ([here][s3-bucket-creation] is the official S3 documentation).

Then, create an access key and secret key for the bucket. This is the key the a8s control plane
will use to interact with the bucket.

After you've created the access key and secret key, you must place the information about the S3
bucket in some files as shown in the following commands. When you'll execute the commands to install
a8s, the content of such files will be used to populate configmaps and secrets that the a8s control
plane will read to be able to upload and download backups from S3. In order to encrypt the backups 
you also have to configure an encryption password. You can do so by inserting your desired 
encryption key into the `deploy/a8s/encryption-password` file. You MUST use the file names shown in
the subsequent commands.

You should also configure an encryption password for the backups.

```shell
echo <bucket-access-key-id> > deploy/a8s/access-key-id # create file that stores the ID of the key

echo <bucket-secret-access-key> > deploy/a8s/secret-access-key # create file that stores the secret value of the key

echo <encryption password> > deploy/a8s/encryption-password # create file that stores password for backup encryption

cp deploy/a8s/backup-store-config.yaml.template config/backup-store-config.yaml # create file with other information about the bucket
```

Then, use an editor to open `deploy/a8s/backup-store-config.yaml` and replace the value:

- of the `container` field with the name of the S3 bucket
- of the `region` field with the name of the region where the bucket is located

All the created files are gitignored so you don't have to worry about committing them by mistake
(since they contain private data).

### Configure Images
The images the frameworks uses to create the Data Service Instances can be configured. A ConfigMap
with the default values is provided at `deploy/a8s/postgresql-images.yaml`. If you need to use different
images, or want to mirror them to an internal repository, you can edit the config, or overwrite it
via Kustomize.

Currently the following images can be configured:

| Key              | Description                                                                |
|------------------|----------------------------------------------------------------------------|
| spiloImage       | Image of spilo, which provides PostgreSQL and patroni for HA               |
| backupAgentImage | Image of the a9s backup agent, which performs logical backups and restores |

Please note: the images will change over time, as we upgrade our framework components. If the 
defaults have been changed, they should be updated when we update the images.

### RBAC

The a8s framework requires multiple ClusterRoles as well as multiple ClusterRoleBindings per
component in order to function properly. These Kubernetes resources are part of the individual
manifests of the a8s framework components (e.g. `postgresql-operator.yaml`) and are automatically
created when the framework is deployed using kustomize. Nevertheless the ClusterRole and
ClusterRoleBinding can be updated based on your specific environmental requirements.

### Install the a8s Control Plane

Just run:

```shell
kubectl apply --kustomize deploy/a8s/
```

This command will create the Kubernetes resources that make up the a8s control plane in the correct
order.

More precisely, it will:

1. Create two namespaces called `a8s-system` and `postgresql-system`.
   The `postgresql-system` namespace is used for the `postgresql-controller-manager`, the rest of
   the a8s framework components (`a8s-backup-controller-manager` and
   `service-binding-controller-manager`) are running in `a8s-system`.
2. Register multiple CustomResourceDefinitions (CRDs)
3. Create three deployments, one for each a8s framework component
4. Create multiple ClusterRoles and ClusterRoleBindings:
- <component_name>-manager-role and <component_name>-manager-rolebinding:  
  provides the a8s framework components access to Kubernetes resources
- <component_name>-metrics-reader:  
  provide access to the metrics endpoint
- <component_name>-proxy-role and <component_name>-proxy-rolebinding:  
  used for access authentication to secure the access to metrics
- postgresql-spilo-role:  
  gives spilo the required permissions to access Kubernetes resources

5. Create one Role and RoleBinding:
- <component_name>-leader-election-role and <component_name>-leader-election-rolebinding:  
  used for communication between multiple controllers of the same type.  
  **Note** The a8s framework is not HA ready, therefore this ClusterRole is currently
  not actively used.

6. Generate and apply multiple Configmaps (e.g. `a8s-backup-store-config`) and Secrets (e.g. 
`a8s-backup-storage-credentials`) that are necessary for the a8s framework in order to function
  properly.

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
3. OpenSearch and OpenSearch Dashboards to query and visualize the logs.

To install them, simply run:

```shell
kubectl apply --kustomize deploy/logging
```

To wait for all components to be up and running, run:

```shell
watch kubectl get pod --namespace a8s-system --selector=a8s.anynines/logging
```

and wait until you see that all the pods (3 + the number of worker nodes of your Kubernetes cluster)
are running (value `1/1` under the `READY` column):

```shell
NAME                                         READY   STATUS    RESTARTS   AGE
a8s-fluentd-aggregator-0                     1/1     Running   0          6m20s
a8s-opensearch-cluster-0                     1/1     Running   0          6m20s
a8s-opensearch-dashboards-648cb7d4f4-6xmq8   1/1     Running   0          6m20s
fluent-bit-jqfgl                             1/1     Running   0          6m20s
```

### Uninstall the Logging Infrastructure

Run:

```shell
kubectl delete --kustomize deploy/logging
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

[s3-bucket-creation]:
    https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html
[cert-manager]: 
    https://cert-manager.io/docs/
[cert-manager-compatibility]:
    https://cert-manager.io/docs/installation/compatibility/
[cert-manager-config]:
    https://cert-manager.io/docs/configuration/
