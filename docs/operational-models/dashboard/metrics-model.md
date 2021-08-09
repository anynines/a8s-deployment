# Metrics dashboard operational model

## Manual Steps

In the following, we are going to deploy a Grafana dashboard that will query the
system metrics that are scraped by Prometheus and display them in a metrics
dashboard.

### Creation

Just as for the Prometheus instance, we will deploy the Grafana dashboard in the
`a8s-system` namespace. In order to create it we need to issue the following
command:

```shell
kubectl apply -f dashboard/a8s-system.yaml
```

First we need to deploy the Grafana dashboard.
It is configured in a way to periodically send queries to the Prometheus
service. By default Grafana does not come with any default dashboard. If we
want to use one we either need to define it ourselves or we can import an
existing one from the [Grafana Dashboards][Grafana Dashboards] page using the
Dashboard ID. In order to access the Grafana dashboard we need a port-forward:
`kubectl port-forward -n a8s-system service/grafana 3000 &` followed by a
`open http://localhost:3000` to actually open the dashboard. The default login
credentials are `admin` for both username and password. If we want to import
pre-build dashboards we need to click on the left bar on `Dashboards` and then
on `Manage`. On the right side we can then click on `Import` and paste the
dashboard ID from the [Grafana Dashboards][Grafana Dashboards] page.

```shell
kubectl apply -f dashboard/grafana-configmap.yaml
kubectl apply -f dashboard/grafana-deployment.yaml
kubectl apply -f dashboard/grafana-service.yaml
```

### Deletion

If you want to get rid of the Grafana dashboard, you can run the following commands:

```shell
kubectl delete -f dashboard/grafana-service.yaml
kubectl delete -f dashboard/grafana-deployment.yaml
kubectl delete -f dashboard/grafana-configmap.yaml
```

[Grafana Dashboards]: https://grafana.com/grafana/dashboards
