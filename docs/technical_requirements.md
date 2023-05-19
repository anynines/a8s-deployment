# Technical Requirements

## General Prerequisites

To try out a8s you will need:

- a running Kubernetes cluster (for the size we recommend considering the
  [estimated resource consumption](#resource-consumption-estimates) of a8s)
- `kubectl` v1.14 or higher pointed to the Kubernetes cluster
- one [StorageClass][storage-class] marked as `default` in the Kubernetes cluster
- one AWS S3 bucket for storing Backups

If you want your data service instances to have IPs reachable from outside the K8s cluster
(by using the feature described
[here](./application-developers/advanced_configuration.md#usage-outside-the-kubernetes-cluster)),
the K8s cluster must support [external load balancers][k8s-load-balancer-services].

To access the included Dashboards (Grafana, OpenSearch Dashboards) it is also
recommended to deploy an Ingress Controller on your cluster. To access the
Dashboards you can then expose the Dashboard Services, for more information on
the process, consult the [Kubernetes Ingress Documentation][k8s-ingress].

The instructions in this repo have been tested on Kubernetes v1.20, v1.21,
v1.22 and v1.23 on minikube, and on Kubernetes v1.22 on EKS, but they should
work with any recent Kubernetes version. Please let us know if you encounter any
issue with other versions.

## Resource Consumption Estimates

The purpose of this section is to provide a rough estimate for the resources
that the components of a8s will consume. These numbers are obtained under only a
minimal load (only a8s and a single PostgreSQL DSI on the cluster) and therefore
can vary heavily depending on the workloads on your cluster, especially the
resource consumption of the logging and metrics components.

| Component                           | CPU (cores) | Memory (MiB)  |
|-------------------------------------|-------------|---------------|
|**a8s Control Plane**
|a8s Backup Controller Manager v0.3.0 | 0.005       | 22            |
|a8s Service Binding Controller v0.2.0 | 0.004       | 22            |
|Postgresql Operator v0.7.0            | 0.004       | 20            |
|*Total*                              | 0.013       | 64            |
|                                     |             |               |
|**a8s Logging Framework**
|Fluent Bit (DaemonSet)               | 0.002 per node | 6 per node|
|FluentD                              | 0.013       | 95            |
|OpenSearch                           | 0.021       | 1000          |
|OpenSearch Dashboards                | 0.002       | 130           |
|                                     |             |               |
|**a8s Metrics**
|Grafana                              | 0.003       | 35            |
|Prometheus                           | 0.045       | 360           |
|                                     |             |               |
|**PostgreSQL DSI**
|Single PostgreSQL Pod (no load)      | 0.003       | 130           |

Likely, you are already running more recent versions of the a8s control plane components than those
that have been used to obtain these numbers. But, we expect the numbers to have changed only
marginally.

[storage-class]: https://kubernetes.io/docs/concepts/storage/storage-classes/
[k8s-ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/
[k8s-load-balancer-services]: https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer
