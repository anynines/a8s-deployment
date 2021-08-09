# Metrics Operational Model

Based on Minikube.

We create a separate namespace for a8s system components such as
the metrics framework. This allows us to isolate a8s Kubernetes objects from
the rest of the cluster.

```shell
kubectl apply -f dashboard/a8s-system.yaml
```

Kubernetes provides out-of-the-box control plane and container metrics in
prometheus format. These metrics can be scraped at the /metrics endpoint at
the Kubernetes API Server.

In order to scrape these logs we will deploy a Prometheus instance that is used
to scrape the system metrics and store them in the internal database.
Furthermore, we are going to deploy a Grafana dashboard that is able to query
the Prometheus database and visualize the metrics in a metrics dashboard.

# Prometheus

## Manual Steps

In the following we will deploy and use Prometheus to scrape and store the
metrics on minikube.

### Creation

First we need to install the Prometheus deployment.
We need to give Prometheus certain permissions to access the /metrics endpoint.
These permissions are defined in the `prometheus-permissions.yaml` file.
Moreover we configure the Prometheus instance with the
`prometheus-configmap.yaml` file that defines the endpoints that will be
scraped as well as the corresponding scrape interval. The prometheus service
that is defined in the `prometheus-service.yaml` is used by the Grafana
dashboard.

```shell
kubectl apply -f metrics/prometheus-permissions.yaml
kubectl apply -f metrics/prometheus-configmap.yaml
kubectl apply -f metrics/prometheus-deployment.yaml
kubectl apply -f metrics/prometheus-service.yaml
```

### Deletion

If you want to get rid of the Prometheus instance, you can run the following
commands:

```shell
kubectl delete -f metrics/prometheus-service.yaml
kubectl delete -f metrics/prometheus-deployment.yaml
kubectl delete -f metrics/prometheus-configmap.yaml
kubectl delete -f metrics/prometheus-permissions.yaml
```
