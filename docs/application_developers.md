# Application Developers Documentation

This file contains documentation specificly meant for application developers.

- [Usage Overview: Deploy and Use a PostgreSQL Instance](#usage-overview-deploy-and-use-a-postgresql-instance)
  - [Provision a PostgreSQL Instance](#provision-a-postgresql-instance)
  - [Bind an Application to the PostgreSQL Instance](#bind-an-application-to-the-postgresql-instance)
  - [Take a Backup of the PostgreSQL Instance](#take-a-backup-of-the-postgresql-instance)
  - [Use a Backup to Restore a PostgreSQL Instance to a Previous State](#use-a-backup-to-restore-a-postgresql-instance-to-a-previous-state)
  - [Visualize the Logs of the PostgreSQL Instance](#visualize-the-logs-of-the-postgresql-instance)
  - [Visualize the Metrics of the PostgreSQL Instance](#visualize-the-metrics-of-the-postgresql-instance)

## Usage Overview: Deploy and Use a PostgreSQL Instance

This section is an overview of how you (an application developer) can use a8s to provision a
PostgreSQL instance, bind an application to it and use it.

The following subsections assume, besides the [General Prerequisites](/docs/platform_operators.md#general-prerequisistes),
that you or a platform operator have installed a8s on the Kubernetes cluster following the
instructions in the section [Install the a8s Control Plane](/docs/platform_operators.md#install-the-a8s-control-plane).

### Provision a PostgreSQL Instance

To provision a PostgreSQL instance, you have to `kubectl apply` a yaml manifest that describes
an API object of the `PostgreSQL` custom kind (which gets installed as part of a8s).

There's an example of such a manifest at
[examples/postgresql-instance.yaml](/examples/postgresql-instance.yaml), so to provision it you can
run:

```shell
kubectl apply --filename examples/postgresql-instance.yaml
```

This will install a 3-replica PostgreSQL streaming replication cluster, where each replica runs in
a Pod. It might take some time for the cluster to be up and running. To know when that has happened,
run:

```shell
watch kubectl get postgresql sample-pg-cluster --output template='{{.status.readyReplicas}}'
```

and wait until the output is equal to "3".

Know that when creating `PostgreSQL` API objects you can specify more fields than those shown in
[examples/postgresql-instance.yaml](/examples/postgresql-instance.yaml). Also, you can dynamically
update most fields. Stay tuned for a complete API reference where we'll detail all the fields.

To delete the PostgreSQL instance, you can run:

```shell
kubectl delete --filename examples/postgresql-instance.yaml
```

### Bind an Application to the PostgreSQL Instance

To make an application running in a Kubernetes Pod use a provisioned PostgreSQL instance, you first
have to create a `ServiceBinding` custom API object (another a8s custom API type).

A `ServiceBinding` always points to a data service instance and represents a user inside that
instance.
This will be the user that your application logs in as when interacting with that data service
instance.

At [examples/service-binding.yaml](/examples/service-binding.yaml) there's the yaml manifest of an
example `ServiceBinding` that points to the PostgreSQL instance you previously deployed. Run:

```shell
kubectl apply --filename examples/service-binding.yaml
```

and wait for it to be implemented by the a8s control plane: run

```shell
watch kubectl get servicebinding sb-sample --output template='{{.status.implemented}}'
```

and wait until the output is "true".

The a8s control plane implemented the `ServiceBinding` by creating a user inside the PostgreSQL
instance. But to log in as that user and start reading/writing data in the database, your
application needs to know:

- the name of the user
- the password of the user
- the database to write to and read from
- the hostname of the PostgreSQL instance

To that end, when you create a service binding named _X_ the a8s control plane creates a Kubernetes
secret called _X-service-binding_ with the following key-val pairs:

- `data.username`: the name of the user
- `data.password`: the password of the user
- `data.database`: the database to write to and read from
- `data.instance_service`: the hostname of the PostgreSQL instance

Notice that currently there's no key with the port of the PostgreSQL instance: that's because it's
always `5432`.

You can verify that such a secret exists by running:

```shell
kubectl get secret sb-sample-service-binding
```

the output should be:

```shell
NAME                        TYPE     DATA   AGE
sb-sample-service-binding   Opaque   3      94m
```

Your application can therefore know all the credentials and information required to connect to the
PostgreSQL instance in two different ways:

- [populating environment variables from the secret's keys][mount-secret-in-env-vars]
- [mounting the secret in a volume][mount-secret-in-volume]

Notice that this means that you have to create the `ServiceBinding` before deploying your app.

Here's an example of a dummy Pod that loads the service binding credentials and simply logs them by
using the first approach (environment variables).

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hello
spec:
  containers:
  - name: hello
    image: busybox
    command: ['sh', '-c', 'echo "PG Username: $PG_USERNAME, PG Password: $PG_PASSWORD" && sleep 3600']
    env:
    - name: "PG_HOST"
      valueFrom:
        secretKeyRef:
          name: sb-sample-service-binding
          key: instance_service
    - name: "PG_USERNAME"
      valueFrom:
        secretKeyRef:
          name: sb-sample-service-binding
          key: username
    - name: "PG_PASSWORD"
      valueFrom:
        secretKeyRef:
          name: sb-sample-service-binding
          key: password
    - name: "PG_PORT"
      value: "5432"
    - name: "PG_DATABASE" # TODO change this to be read from secret after we fix service binding
    # controller
      value: "demo"
```

There's no limit to how many `ServiceBindings` can point to the same data service instance.
When a data service instance is deleted, all `ServiceBindings` that point to it are also
automatically deleted (i.e. the API objects will be deleted from the Kubernetes API).

When you want to delete a service binding, just run:

```shell
kubectl delete servicebinding <service-binding name>
```

### Take a Backup of the PostgreSQL Instance

To backup a data service instance, you have to create a custom API object of kind `Backup` (a
custom kind which is part of a8s). In its fields, a `Backup` API object points to the data service
instance to backup.

At [examples/backup.yaml](/examples/backup.yaml) there's the yaml manifest of an example `Backup`
that points to the PostgreSQL instance that you previously deployed. Run:

```shell
kubectl apply --filename examples/backup.yaml
```

The a8s control plane will react by taking the backup and uploading it to a cloud storage (currently
only S3) that must have been configured by the platform operator when he installed a8s on your
cluster. This might take some time, to learn when the backup has completed, run:

```shell
watch kubectl get backup backup-sample --output template='{{.status.condition.type}}'
```

and wait until the output is "Succeeded".

Stay tuned for a complete reference of all the fields that you can configure in a Backup API object.

When you want to delete a `Backup`, run:

```shell
kubectl delete backup <backup-name>
```

### Use a Backup to Restore a PostgreSQL Instance to a Previous State

If you want to restore a data service instance to a previous state it had, you can restore it from
a previously taken backup (as shown in section
[Take a Backup of the PostgreSQL Instance](#take-a-backup-of-the-postgresql-instance)).

To do that, you have to create a custom API object of kind `Recovery` (a custom kind which is part
of a8s). A `Recovery` API object fields identify the `Backup` API object to use to perform the
restore. The `Recovery` will always be performed on the data service instance from which the backup
was taken. Stay tuned for a complete reference of all the fields of `Recovery` API objects.

At [examples/recovery.yaml](/examples/recovery.yaml) there's the yaml manifest of an example
`Recovery` that points to the PostgreSQL instance that you previously deployed. Run:

```shell
kubectl apply --filename examples/recovery.yaml
```

The a8s control plane will react by downloading the relevant backup and using it to restore the data
service instance. This might take some time, to learn when the recovery has completed, run:

```shell
watch kubectl get recovery recovery-sample --output template='{{.status.condition.type}}'
```

and wait until the output is "Succeeded".

When you want to delete a `Recovery`, run:

```shell
kubectl delete recovery <recovery-name>
```

### Visualize the Logs of the PostgreSQL Instance

Application developers should be aware that all pods with the label field `app`
will be adjusted within OpenSearch to have the label `app.kubernetes.io/name`.
This pattern conforms with the [recommended labels][common-labels] expressed
officially in the Kubernetes documentation.

When installing the a8s platform, the platform operator had the option to install components to
collect and visualize the logs of the data service instances (as shown in section
[(Optional) Install the Logging Infrastructure](/docs/platform_operators.md#optional-install-the-logging-infrastructure)).
Among them, there's an OpenSearch Dashboards (that runs in a Pod) that you can use to view the logs of the
PostgreSQL instance that you previously deployed.

How you access the dashboard depends on the specifics of your cluster. In a production
environment you would use an [Ingress][kubernetes-ingress], here we'll just use
[port forwarding][kubernetes-port-forwarding].

Run:

```shell
kubectl port-forward services/a8s-opensearch-dashboards 5601:443 -n a8s-system
```

Then, open the OpenSearch dashboard in your browser.

```shell
open http://localhost:5601
```

![OpenSearchDashboards1](/pics/opensearchdashboards/1.png)

Select `Add data` and then click on the ≡ icon in the top left hand corner. In the menu select `Stack management` in the `Management` section.

![OpenSearchDashboards2](/pics/opensearchdashboards/2.png)

Select `Index Patterns`.

![OpenSearchDashboards3](/pics/opensearchdashboards/3.png)

Click on `Create Index pattern`.

![OpenSearchDashboards4](/pics/opensearchdashboards/4.png)

Create an index pattern for `logstash-*`. And click `> Next step`

![OpenSearchDashboards5](/pics/opensearchdashboards/5.png)

Select `@timestamp` as a time filter field name. And then click `Create index pattern`.

![OpenSearchDashboards6](/pics/opensearchdashboards/6.png)

Go back to the discover tab.

![OpenSearchDashboards7](/pics/opensearchdashboards/7.png)

The logs will be available to interact using your new filter.

![OpenSearchDashboards8](/pics/opensearchdashboards/8.png)

### Visualize the Metrics of the PostgreSQL Instance

When installing the a8s platform, the platform operator had the option to install components to
scrape and visualize the metrics of the data service instances (as shown in section
[(Optional) Install the Metrics Infrastructure](/docs/platform_operators.md#optional-install-the-metrics-infrastructure)).
Among them, there's a Grafana dashboard (that runs in a Pod) and you can use to view the metrics
of the PostgreSQL instance that you previously deployed.

How you access the Grafana dashboard depends on the specifics of your cluster. In a production
environment you would use an [Ingress][kubernetes-ingress], here we'll just use
[port forwarding][kubernetes-port-forwarding] to access the dashboard.

The following images show how to access the Grafana dashboard as well as import a pre-build
dashboard to visualize the logs.

In order to access the Grafana dashboard we need a port-forward to the Grafana
service:

Run:

```shell
kubectl port-forward service/grafana 3000:3000 --namespace=a8s-system
```

Open the Grafana dashboard by issuing:

```shell
open http://localhost:3000
```

Log into the dashboard by using `admin` as a username as well as the password.
Afterwards you need to import a dashboard in order to visualize the metrics that are scraped by the
Prometheus instance.

![Grafana1](/pics/grafana/1.png)

Go to the Dashboards section in the left menu.

![Grafana2](/pics/grafana/2.png)

Then go to the Manage page.

![Grafana3](/pics/grafana/3.png)

Click on Import on the right hand side.

![Grafana4](/pics/grafana/4.png)
 
Then Insert `8588` as the Dashboard ID and click on Load.

![Grafana5](/pics/grafana/5.png)

Choose Prometheus as the data source.

![Grafana6](/pics/grafana/6.png)

Now the imported metrics dashboard should visualize some of the metrics
that are scraped by the Prometheus instance.

[storage-class]: https://kubernetes.io/docs/concepts/storage/storage-classes/
[s3-bucket-creation]: https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html
[mount-secret-in-env-vars]: https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables
[mount-secret-in-volume]: https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod
[kubernetes-ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/
[kubernetes-port-forwarding]: https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/
[common-labels]: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
