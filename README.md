# a8s-deployment

**WARNING: This repo is WIP. We will cleanup, squash and force commit `main` in
the future.**

## minikube on macOS

### Prerequisites

```shell
minikube start
minikube addons enable registry

docker run --rm -it --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"
```

### Dashboard

#### Install Elasticsearch

```shell
kubectl apply -f dashboard/a8s-system.yaml
kubectl apply -f dashboard/elasticsearch_svc.yaml
kubectl apply -f dashboard/elasticsearch_statefulset.yaml
kubectl rollout status statefulset/es-cluster --namespace a8s-system
```

#### Install Kibana

```shell
kubectl apply -f dashboard/kibana.yaml
kubectl rollout status deployment/kibana --namespace a8s-system
```

### Logging

#### Install Fluent Bit DaemonSet as a log forwarder

```shell
kubectl apply -f logging/fluent-bit-daemonset-permissions.yaml
kubectl apply -f logging/fluent-bit-daemonset-configmap-forward-minikube.yaml
kubectl apply -f logging/fluent-bit-daemonset-forward-minikube.yaml
```

#### Install Fluentd statefulset as a log aggregator

```shell
kubectl apply -f logging/fluentd-aggregator-configmap.yaml
kubectl apply -f logging/fluentd-aggregator-service.yaml
kubectl apply -f logging/fluentd-aggregator-statefulset.yaml
```

#### Using Dashboard

First, get the Kibana pod name

```shell
kibana=$(kubectl get pod -l app=kibana --namespace a8s-system | grep kibana | awk -F ' ' '{print $1}')
```

Use port-forward to connect to the pod. This is just for testing purposes on
Minikube.

```shell
kubectl port-forward $kibana 5601:5601 --namespace=a8s-system
```

Open the Kibana dashboard in Browser link in browser.

```shell
open http://localhost:5601
```

![Kibana1](operational-models/images/kibana/1.png)

Go to discover in the top left hand corner.

![Kibana2](operational-models/images/kibana/2.png)

Create an index pattern for `logstash-*`. And click `> Next step`

![Kibana3](operational-models/images/kibana/3.png)

Select `@timestamp` as a time filter field name. And then click
`Create index pattern`.

![Kibana4](operational-models/images/kibana/4.png)

Go back to the discover tab.

![Kibana5](operational-models/images/kibana/5.png)

The logs will be available to interact using your new filter.

![Kibana6](operational-models/images/kibana/6.png)

#### Delete Fluent Bit DaemonSet Setup

```shell
kubectl delete -f logging/fluent-bit-daemonset-elasticsearch-minikube.yaml
kubectl delete -f logging/fluent-bit-daemonset-configmap-elasticsearch-minikube.yaml
kubectl delete -f logging/fluent-bit-daemonset-permissions.yaml
```

## a9s PaaS

### Prerequisites

- a9s Kubernetes instance

### Logging

#### Install Fluentd DaemonSet

```shell
kubectl apply -f logging/fluentd-daemonset-permissions.yaml
kubectl apply -f logging/fluentd-daemonset-syslog-a9s-kubernetes.yaml
```
