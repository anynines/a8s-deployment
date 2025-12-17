# Install the a8s Control Plane

- [Install the a8s Control Plane](#install-the-a8s-control-plane)
  - [Prerequisites](#prerequisites)
    - [Configure Backups Store](#configure-backups-store)
    - [Configure S3-Compatible Backups Store](#configure-s3-compatible-backups-store)
  - [Configure Images](#configure-images)
  - [Install the a8s Control Plane](#install-the-a8s-control-plane-1)
    - [Using Static Manifests](#using-static-manifests)
      - [Install the cert-manager](#install-the-cert-manager)
      - [Install the Control Plane with Manifests](#install-the-control-plane-with-manifests)
    - [Using the OLM](#using-the-olm)
      - [Install the OLM](#install-the-olm)
      - [Install the Control Plane with OLM](#install-the-control-plane-with-olm)
      - [Uninstalling the Control Plane](#uninstalling-the-control-plane)
    - [Using Helm Charts](#using-helm-charts)
      - [Installing with Helm](#install-components-with-helm)
      - [Install All Components with Helm](#install-all-components-with-helm)
      - [Install the PostgreSQL Operator](#install-the-postgresql-operator)
      - [Install the Service Binding Controller](#install-the-service-binding-controller)
      - [Install the Backup Manager](#install-the-backup-manager)
  - [(Optional) Install the Logging Infrastructure](#optional-install-the-logging-infrastructure)
    - [Virtual Memory Usage](#virtual-memory-usage)
      - [Disabling Virtual Memory Usage](#disabling-virtual-memory-usage)
  - [Uninstall the Logging Infrastructure](#uninstall-the-logging-infrastructure)
  - [(Optional) Install the Metrics Infrastructure](#optional-install-the-metrics-infrastructure)
  - [Uninstall the Metrics Infrastructure](#uninstall-the-metrics-infrastructure)
- [Building and Publishing Helm Charts](building_helm_charts.md)

## Prerequisites

### Configure Backups Store

a8s supports taking backups of data service instances (DSIs). Currently, the backups are stored in
an AWS S3 bucket, so before installing a8s **you must create an AWS S3 bucket that a8s will use to
store backups** ([here][s3-bucket-creation] is the official S3 documentation).

Then, create a secret access key for the bucket. This is the key the a8s control plane will use to
interact with the bucket.

After you've created the access key, you must place the information about the S3
bucket in some files as shown in the following commands. When you'll execute the commands to install
a8s, the content of such files will be used to populate configmaps and secrets that the a8s control
plane will read to be able to upload and download backups from S3. In order to encrypt the backups
you also have to configure an encryption password. You can do so by inserting your desired
encryption password into the `deploy/a8s/backup-config/encryption-password`
file. You MUST use the file names shown in the subsequent commands.

```shell
# create file that stores the ID of the key
echo -n <bucket-access-key-id> > deploy/a8s/backup-config/access-key-id

# create file that stores the secret value of the key
echo -n <bucket-secret-access-key> > deploy/a8s/backup-config/secret-access-key

# create file that stores password for backup encryption
echo -n <encryption password> > deploy/a8s/backup-config/encryption-password

# create file with other information about the bucket
cp deploy/a8s/backup-config/backup-store-config.yaml.template deploy/a8s/backup-config/backup-store-config.yaml 
```

Then, use an editor to open `deploy/a8s/backup-config/backup-store-config.yaml` and replace the value:

- of the `container` field with the name of the S3 bucket
- of the `region` field with the name of the region where the bucket is located

All the created files are gitignored so you don't have to worry about committing them by mistake
(since they contain private data).

### Configure S3-Compatible Backup Store (Optional)

Alternatively, the framework supports taking backups to any S3-compatible backup store, which can be useful for local testing. This requires installing and configuring a local S3-compatible backup store, such as MinIO, separately.

```shell
# create file that stores the ID of the key
echo -n <bucket-access-key-id> > deploy/a8s/backup-config/access-key-id

# create file that stores the secret value of the key
echo -n <bucket-secret-access-key> > deploy/a8s/backup-config/secret-access-key

# create file that stores password for backup encryption
echo -n <encryption password> > deploy/a8s/backup-config/encryption-password

# create file with other information about the bucket
cp deploy/a8s/backup-config/backup-store-config.yaml.template deploy/a8s/backup-config/backup-store-config.yaml
```

Next, use an editor to open `deploy/a8s/backup-config/backup-store-config.yaml` and replace the value:

- Of the `container` field with the name of the S3 bucket.
- Add `path_style` set to `True`.
- Add `endpoint` with the URL for the S3-compatible service, for example, `http://test-hl.default.svc.cluster.local:9000`.

All the created files are gitignored, so you don't have to worry about committing them by mistake (since they contain private data).

## Configure Images

The images the framework uses to create the Data Service Instances can be configured. A ConfigMap
with the default values is provided at
`deploy/a8s/manifests/postgresql-images.yaml`. If you need to use different
images, or want to mirror them to an internal repository, you can edit the
config, or overwrite it
via Kustomize.

Currently the following images can be configured:

| Key              | Description                                                                |
|------------------|----------------------------------------------------------------------------|
| spiloImage       | Image of spilo, which provides PostgreSQL and patroni for HA               |
| backupAgentImage | Image of the a9s backup agent, which performs logical backups and restores |

Please note: The images will change over time, as we upgrade our framework
components. If the defaults have been changed, they should be updated when we
update the images.

Also, your changes will be overwritten when deploying with the OLM and during an
update. If you need to edit the configMap, reapply it when you deployed or
updated the framework. In this case you might want to disable automatic updates.

> WARNING: Issues may occure when running spilo with the MobilityDB Postgresql extension on arm-based systems. The latest spilo images natively support both amd and arm architectures, which is not the case for MobilityDB.

## Install the a8s Control Plane

The a8s Control Plane can be deployed with the help of the static manifests you
can find under `/deploy/a8s/manifests` or with the help of the [Operator
Lifecycle Manager (OLM)][olm].

While the manifest method is easy to use, it does not come with automatic
updates or lifecycle management of the framework, so we encourage you to use the
OLM.

### Using Static Manifests

#### Install the cert-manager

The a8s framework relies on the [cert-manager][cert-manager] to generate
TLS certificates, therefore you will first have to install it on your cluster.

> Please check the [cert-manager cloud compatibility page][cert-manager-compatibility] to
> ensure your Kubernetes cluster meets all the requirements to run the cert-manager.

In general there are a multitude of [installation
options](https://cert-manager.io/docs/installation/) supported by the
cert-manager, for this guide we will only describe how to setup a basic
deployment, since that suffices for a8s. For a production grade
deployment please consult the [documentation][cert-manager].

To setup a basic deployment, use:

```shell
kubectl apply --kustomize deploy/cert-manager
```

This will install all the cert-manager components that a8s needs. Know that it might take some time
for the components to get up and running (roughly we've experienced 80 secs for a 3-node EKS
cluster), if you install a8s before that has happened things won't work.

> Currently, the few parts of a8s that require TLS use self-signed certificates. If instead you want
> to set up a proper Certificate Authority, please check out the
> [configuration pages][cert-manager-config], where you can find instructions on how to do that.

#### Install the Control Plane with Manifests

Just run:

```shell
kubectl apply --kustomize deploy/a8s/manifests
```

This command will create the Kubernetes resources that make up the a8s control plane in the correct
order.

More precisely, it will:

1. Create a namespace called `a8s-system`.
   The a8s framework components `postgresql-controller-manager`, `a8s-backup-controller-manager` and
   `service-binding-controller-manager` will be running in `a8s-system` namespace.
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

5. Create one Role and RoleBinding for each a8s framework component:

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
```

the output of the command should be similar to:

```shell
NAME                                 READY   UP-TO-DATE   AVAILABLE   AGE
a8s-backup-controller-manager        1/1     1            1           105s
service-binding-controller-manager   1/1     1            1           105s
postgresql-controller-manager        1/1     1            1           105s
```

#### Uninstall the Control Plane with Manifests

To uninstall the control plane, use:

```shell
kubectl delete --kustomize deploy/a8s/manifests
```

This will delete the Kubernetes resources that make up the a8s control plane and their associated CRDs.

To uninstall the cert-manager components, use:

```shell
kubectl delete --kustomize deploy/cert-manager
```

### Using the OLM

#### Install the OLM

If you have the operator-sdk CLI already installed, you can use

```shell
operator-sdk olm install
```

to install the OLM components to your cluster. Alternatively, you can follow the
[official
instructions](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md).

#### Install the Control Plane with OLM

To install the a8s control plane use:

```shell
kubectl apply --kustomize deploy/a8s/olm
```

to apply all OLM resources necessary. In more detail, this will create:

- the namespace `a8s-system`.
- a `CatalogSource` referencing the a8s catalog which contains references to
  our operators
- an `OperatorGroup` a8s-operators linked to the a8s-system namespace, which can
  be used to adjust general permission for all operators in that group. You can
  find more information on that subject
  [here](https://docs.openshift.com/container-platform/4.8/operators/understanding/olm/olm-understanding-operatorgroups.html).
- a `Subscription` to the a8s postgresql-operator. A Subscription indicates your
  desire to have the operator installed to the cluster, the OLM will then fetch
  the bundle of the PostgreSQL operator and its dependencies, which includes
  the a8s-backup-manager and a8s-service-binding-controller. These bundles then
  contain the instructions for the OLM to create the same resources as explained
  in the manifest section.

Additionally, the kustomization creates the secret and configMap for the backup
bucket configuration.

#### Uninstalling the Control Plane with OLM

To uninstall the control plane use:

```shell
kubectl delete --kustomize deploy/a8s/olm
```

This will delete the backup credentials, the subscriptions and therefore
also the control plane deployment, it does not delete the CRDs from the cluster.
The OLM keeps the CRDs because the deletion would cause also the deletion of the
CRs and therefore all instances and also backup objects. In the OLM
documentation it is therefore stated that such a step should only be taken
deliberately by a user. You can delete the CRDs using:

```shell
kubectl delete crd recoveries.backups.anynines.com\
    backups.backups.anynines.com\
    postgresqls.postgresql.anynines.com\
    servicebindings.servicebindings.anynines.com
```

### Using Helm Charts

You can deploy a8s components using Helm charts. Charts are available in two ways:

1. **From the repository** (recommended): Published charts in AWS S3
2. **Locally**: Charts under `deploy/a8s/charts/` in this repository

Published charts are available at:

- PostgreSQL Operator: `https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/postgresql-operator`
- Backup Manager: `https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/backup-manager`
- Service Binding Controller: `https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/service-binding-controller`

This approach is useful if you want to manage specific components independently or integrate a8s with your existing Helm-based infrastructure.

#### Install Components with Helm

Each chart is self-contained and can be deployed independently from either the repository or locally.

#### Install All Components with Helm

To deploy all three a8s components (PostgreSQL Operator, Service Binding Controller, and Backup Manager)
together in a single command, similar to the `kubectl apply --kustomize` approach:

1. **Prerequisites**:

   - Ensure cert-manager is installed (see [Install the cert-manager](#install-the-cert-manager))
   - Prepare AWS S3 credentials (access key ID, secret access key, encryption password)

2. **Install from published charts** (recommended):

   First, add the Helm repositories:

   ```shell
   helm repo add postgresql-operator https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/postgresql-operator
   helm repo add backup-manager https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/backup-manager
   helm repo add service-binding-controller https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/service-binding-controller
   helm repo update
   ```

   Then install the charts:

   ```shell
   helm install postgresql-operator \
     postgresql-operator/postgresql-operator \
     --namespace a8s-system \
     --create-namespace && \
   helm install backup-manager \
     backup-manager/backup-manager \
     --namespace a8s-system \
     --set backupStorageConfig.secret.create=true \
     --set backupStorageConfig.secret.accessKeyId=YOUR_AWS_ACCESS_KEY_ID \
     --set backupStorageConfig.secret.secretAccessKey=YOUR_AWS_SECRET_ACCESS_KEY \
     --set backupStorageConfig.secret.encryptionPassword=YOUR_ENCRYPTION_PASSWORD && \
   helm install service-binding-controller \
     service-binding-controller/service-binding-controller \
     --namespace a8s-system
   ```

   Or **install from local charts**:

   First, configure the values file at `deploy/a8s/charts/a8s-controlplane-values.yaml` and update the required values:

   - `backupManager.backupStorageConfig.secret.accessKeyId`: Your AWS access key
   - `backupManager.backupStorageConfig.secret.secretAccessKey`: Your AWS secret key
   - `backupManager.backupStorageConfig.secret.encryptionPassword`: Your encryption password
   - `backupManager.backupStorageConfig.configMap.config.cloud_configuration.container`: Your S3 bucket name
   - `backupManager.backupStorageConfig.configMap.config.cloud_configuration.region`: Your S3 region

   Then install:

   ```shell
   helm install postgresql-operator ./deploy/a8s/charts/a8s-postgresql-operator \
     --namespace a8s-system \
     --create-namespace \
     -f deploy/a8s/charts/a8s-controlplane-values.yaml && \
   helm install backup-manager ./deploy/a8s/charts/a8s-backup-manager \
     --namespace a8s-system \
     -f deploy/a8s/charts/a8s-controlplane-values.yaml && \
   helm install service-binding-controller ./deploy/a8s/charts/a8s-service-binding-controller \
     --namespace a8s-system \
     -f deploy/a8s/charts/a8s-controlplane-values.yaml
   ```

   For image configurations and all available options, see the individual chart values files:
   - [PostgreSQL Operator values](../../deploy/a8s/charts/a8s-postgresql-operator/values.yaml)
   - [Backup Manager values](../../deploy/a8s/charts/a8s-backup-manager/values.yaml)
   - [Service Binding Controller values](../../deploy/a8s/charts/a8s-service-binding-controller/values.yaml)

3. **Verify all components are running**:

   ```shell
   kubectl get deployment -n a8s-system
   ```

   All three deployments should show `1/1` under the `READY` column:

   ```shell
   NAME                                          READY   UP-TO-DATE   AVAILABLE   AGE
   a8s-postgresql-operator-controller-manager    1/1     1            1           2m
   a8s-backup-manager-controller-manager         1/1     1            1           2m
   a8s-service-binding-controller-manager        1/1     1            1           2m
   ```

4. **Uninstall all components**:

   ```shell
   helm uninstall postgresql-operator --namespace a8s-system && \
   helm uninstall backup-manager --namespace a8s-system && \
   helm uninstall service-binding-controller --namespace a8s-system
   ```

   To also delete the CRDs:

   ```shell
   kubectl delete crd postgresqls.postgresql.anynines.com \
     backups.backups.anynines.com \
     restores.backups.anynines.com  \
     backuppolicies.backups.anynines.com  \
     servicebindings.servicebindings.anynines.com
   ```

#### Install the PostgreSQL Operator

To install the a8s PostgreSQL Operator using Helm:

1. **Prerequisites**: Ensure cert-manager is installed on your cluster (see
[Install the cert-manager](#install-the-cert-manager) section above).

2. **Install from the published chart** (recommended):

   ```shell
   helm install a8s-postgresql-operator \
     oci://public.ecr.aws/w5n9a2g2/anynines/klutch/charts/postgresql-operator \
     --version 0.1.0 \
     --namespace a8s-system \
     --create-namespace
   ```

   Or **use the local Helm chart**:

   ```shell
   helm install a8s-postgresql-operator ./deploy/a8s/charts/a8s-postgresql-operator \
     --namespace a8s-system \
     --create-namespace
   ```

3. **Customize image tag** (if using private or mirrored images):

   ```shell
   helm install a8s-postgresql-operator \
     oci://public.ecr.aws/w5n9a2g2/anynines/klutch/charts/postgresql-operator \
     --version 0.1.0 \
     --namespace a8s-system \
     --create-namespace \
     --set image.tag=your-tag
   ```

   For PostgreSQL image configuration (Spilo, backup agent), edit the values:

   ```shell
   helm install a8s-postgresql-operator \
     oci://public.ecr.aws/w5n9a2g2/anynines/klutch/charts/postgresql-operator \
     --version 0.1.0 \
     --namespace a8s-system \
     --create-namespace \
     --set postgresqlImages.spiloImage=your-registry.com/spilo:latest \
     --set postgresqlImages.backupAgentImage=your-registry.com/backup-agent:latest
   ```

   See [values.yaml](../../deploy/a8s/charts/a8s-postgresql-operator/values.yaml) for all
   available configuration options.

4. **Verify the deployment**:

   ```shell
   kubectl get deployment -n a8s-system a8s-postgresql-operator-controller-manager
   ```

   The deployment should show `1/1` under the `READY` column.

5. **Uninstall** (if needed):

   ```shell
   helm uninstall a8s-postgresql-operator --namespace a8s-system
   ```

   To also delete the CRD, run:

   ```shell
   kubectl delete crd postgresqls.postgresql.anynines.com
   ```

#### Install the Service Binding Controller

To install the a8s Service Binding Controller using Helm:

1. **Prerequisites**: Ensure cert-manager is installed on your cluster (see
[Install the cert-manager](#install-the-cert-manager) section above).

2. **Install from the published chart** (recommended):

   ```shell
   helm install a8s-service-binding-controller \
     oci://public.ecr.aws/w5n9a2g2/anynines/klutch/charts/service-binding-controller \
     --version 0.1.0 \
     --namespace a8s-system \
     --create-namespace
   ```

   Or **use the local Helm chart**:

   ```shell
   helm install a8s-service-binding-controller ./deploy/a8s/charts/a8s-service-binding-controller \
     --namespace a8s-system \
     --create-namespace
   ```

3. **Customize the deployment** by overriding values:

   ```shell
   helm install a8s-service-binding-controller \
     oci://public.ecr.aws/w5n9a2g2/anynines/klutch/charts/service-binding-controller \
     --version 0.1.0 \
     --namespace a8s-system \
     --create-namespace \
     --set controllerConfig.enable-integration=postgresql \
     --set replicaCount=1
   ```

   See [values.yaml](../../deploy/a8s/charts/a8s-service-binding-controller/values.yaml) for all
   available configuration options.

4. **Verify the deployment**:

   ```shell
   kubectl get deployment -n a8s-system a8s-service-binding-controller-manager
   ```

   The deployment should show `1/1` under the `READY` column.

5. **Uninstall** (if needed):

   ```shell
   helm uninstall a8s-service-binding-controller --namespace a8s-system
   ```

   This will remove the Service Binding Controller deployment and RBAC resources, but will retain
   the CustomResourceDefinition (CRD). To also delete the CRD, run:

   ```shell
   kubectl delete crd servicebindings.servicebindings.anynines.com
   ```

#### Install the Backup Manager

To install the a8s Backup Manager using Helm:

1. **Prerequisites**:

   - Ensure cert-manager is installed (see [Install the cert-manager](#install-the-cert-manager))
   - Prepare AWS S3 credentials (access key ID, secret access key, encryption password)
   - Note the S3 bucket name and region

2. **Configure backup storage credentials**:

   Option A: Create the backup storage secret externally and reference it:

   ```shell
   kubectl create secret generic a8s-backup-storage-credentials \
     --from-literal=access-key-id=YOUR_AWS_ACCESS_KEY_ID \
     --from-literal=secret-access-key=YOUR_AWS_SECRET_ACCESS_KEY \
     --from-literal=encryption-password=YOUR_ENCRYPTION_PASSWORD \
     -n a8s-system
   ```

   Then install with the backup-manager from the published chart:

   ```shell
   helm install a8s-backup-manager \
     oci://public.ecr.aws/w5n9a2g2/anynines/klutch/charts/backup-manager \
     --version 0.1.0 \
     --namespace a8s-system \
     --set backupStorageConfig.secret.create=false
   ```

   Or use the local chart:

   ```shell
   helm install a8s-backup-manager ./deploy/a8s/charts/a8s-backup-manager \
     --namespace a8s-system \
     --set backupStorageConfig.secret.create=false
   ```

   Option B: Provide credentials directly to Helm (use with caution, best for testing):

   ```shell
   helm install a8s-backup-manager \
     oci://public.ecr.aws/w5n9a2g2/anynines/klutch/charts/backup-manager \
     --version 0.1.0 \
     --namespace a8s-system \
     --set backupStorageConfig.secret.create=true \
     --set backupStorageConfig.secret.accessKeyId=YOUR_AWS_ACCESS_KEY_ID \
     --set backupStorageConfig.secret.secretAccessKey=YOUR_AWS_SECRET_ACCESS_KEY \
     --set backupStorageConfig.secret.encryptionPassword=YOUR_ENCRYPTION_PASSWORD
   ```

3. **Customize the backup storage configuration**:

   ```shell
   helm install a8s-backup-manager \
     oci://public.ecr.aws/w5n9a2g2/anynines/klutch/charts/backup-manager \
     --version 0.1.0 \
     --namespace a8s-system \
     --set backupStorageConfig.configMap.config.cloud_configuration.provider=AWS \
     --set backupStorageConfig.configMap.config.cloud_configuration.container=my-backup-bucket \
     --set backupStorageConfig.configMap.config.cloud_configuration.region=eu-central-1 \
     --set backupStorageConfig.secret.create=false
   ```

   Or for S3-compatible storage (MinIO):

   ```shell
   helm install a8s-backup-manager ./deploy/a8s/charts/a8s-backup-manager \
     --namespace a8s-system \
     -f - <<EOF
   backupStorageConfig:
     configMap:
       name: a8s-backup-store-config
       config:
         cloud_configuration:
           provider: "S3"
           container: "my-bucket"
           region: "us-east-1"
           endpoint: "http://minio.default.svc.cluster.local:9000"
           path_style: true
     secret:
       create: false
   EOF
   ```

   See [values.yaml](../../deploy/a8s/charts/a8s-backup-manager/values.yaml) and
   [README.md](../../deploy/a8s/charts/a8s-backup-manager/README.md) for all available options.

4. **Verify the deployment**:

   ```shell
   kubectl get deployment -n a8s-system a8s-backup-manager-controller-manager
   ```

   The deployment should show `1/1` under the `READY` column.

5. **Uninstall** (if needed):

   ```shell
   helm uninstall a8s-backup-manager --namespace a8s-system
   ```

   To also delete the CRD, run:

   ```shell
   kubectl delete crd backups.backups.anynines.com restores.backups.anynines.com  \
     backuppolicies.backups.anynines.com 
   ```

## (Optional) Install the Logging Infrastructure

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

### Virtual Memory Usage

OpenSearch heavily rely on virtual memory usage (so `mmap`). When applying the
logging framework, you might have to adjust the `mmap limit` on your nodes,
otherwise the OpenSearch pods will fail, with the error message:

```
ERROR: [1] bootstrap checks failed [1]: max virtual memory areas vm.max_map_count [65530] is too low, increase to at least [262144]
```

If you are running the framework on something like `minikube` or `kind` using
Docker, this does not apply. Otherwise, you can find out more on how to adjust
the virtual memory in the
[OpenSearch](https://opensearch.org/docs/latest/opensearch/install/important-settings/)
documentation on this topic.

#### Disabling Virtual Memory Usage

> **Important Note:**
> The
> [documentation](https://www.elastic.co/guide/en/cloud-on-k8s/current/k8s-virtual-memory.html)
> explicitly warns you to not use this setup in production
> grade workloads.

If you just want to experiment with the framework, or you do not have privileged
access to the nodes, to adjust the virtual memory, you can
disable the usage in the OpenSearch configuration. For that set the `allow_mmap`
flag in the OpenSearch configuration, located in
`deploy/logging/dashboard/config/opensearch.yaml`, to false by appending:

```yml
node:
  store:
    allow_mmap: false
```

After applying the change and restarting the `a8s-opensearch-cluster-0` pod by
using

```shell
kubectl delete pod a8s-opensearch-cluster-0 -n a8s-system
```

OpenSearch should now work without issues.

## Uninstall the Logging Infrastructure

Run:

```shell
kubectl delete --kustomize deploy/logging
```

## (Optional) Install the Metrics Infrastructure

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

and wait until you see that the Prometheus and Grafana Pods are running
(value `1/1` under the `READY` column):

```shell
NAME                                        READY   STATUS    RESTARTS   AGE
a8s-system    pod/grafana-64c89f57f7-v7wlp                1/1     Running   0          111s
a8s-system    pod/prometheus-deployment-87cc8fb88-225v4   1/1     Running   0          111s
```

## Uninstall the Metrics Infrastructure

Run:

```shell
kubectl delete --recursive --filename deploy/metrics/
```

[s3-bucket-creation]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html
[cert-manager]: https://cert-manager.io/docs/
[cert-manager-compatibility]: https://cert-manager.io/docs/installation/compatibility/
[cert-manager-config]: https://cert-manager.io/docs/configuration/
[olm]: https://olm.operatorframework.io/
