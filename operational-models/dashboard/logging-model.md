# Logging dashboard operational model

## Manual steps

### Creation

Currently, we use Elasticsearch and Kibana. Due to licensing, this will be
changed in future. OpenSearch provides a fork of Elasticsearch and Kibana but
the development is still in early beta. We will watch the development and
switch out the Elasticsearch statefulset and Kibana Deployment for the
OpenSearch forks in due course.

```bash
kubectl apply -f dashboard/a8s-system.yaml
```

We create a separate [Namespace][namespace] for a8s system components such as
the logging framework. This allows us to isolate a8s Kubernetes objects from
the rest of the cluster.

```bash
kubectl apply -f dashboard/elasticsearch_svc.yaml
```

A headless Service is a [Service][service] with a service IP but instead of 
load-balancing it will return the IPs of our associated Pods. This allows us to
interact directly with the Pods instead of a proxy. This is useful for taking
advantage of the sticky identity of each pod in the [StatefulSet][statefulset]
rather than any of the pods at random as is the case with a normal Service.

```bash
kubectl apply -f dashboard/elasticsearch_statefulset.yaml
```

Elasticsearch is the distributed, RESTful search and analytics engine which we
use to store, search and manage our cluster's logs. The  Elasticsearch pods 
are deployed as a part of a [StatefulSet][statefulset] since they require
persistent storage for the logs. This also allows us to span the pods over
multiple availability zones for high availability in order to limit possible
downtime.

```bash
kubectl apply -f dashboard/kibana.yaml
```

Kibana is a window into Elasticsearch. It provides a browser-based analytics
and search dashboard. We deploy it as a Kubernetes [Deployment][deployment]
since we don't require storage for Kibana.

[namespace]: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
[service]: https://kubernetes.io/docs/concepts/services-networking/service/
[statefulSet]: https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/
[deployment]: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
