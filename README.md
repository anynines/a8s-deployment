# a8s-deployment

## minikube on macOS

### Prerequisites

```shell
minikube start
minikube addons enable registry

docker run --rm -it --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"
```

### Logging

WIP: ip is hardcoded to RG's local setup

#### Install Fluentd DaemonSet

```shell
kubectl apply -f logging/fluentd-daemonset-permissions.yaml
kubectl apply -f logging/fluentd-daemonset-syslog.yaml
```

##### Delete Fluentd DaemonSet Setup

```shell
kubectl delete -f logging/fluentd-daemonset-syslog.yaml
```
