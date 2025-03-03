<!--kubestellar-check-syncers-start-->
If you are unable to see the namespace 'my-namespace' or the deployment 'my-first-kubestellar-deployment' you can view the logs for the KubeStellar syncer on the **ks-edge-cluster1** Kind cluster:

```shell
KUBECONFIG=~/.kube/config kubectl config use-context ks-edge-cluster1
ks_ns_edge_cluster1=$(KUBECONFIG=~/.kube/config kubectl get namespaces \
    -o custom-columns=:metadata.name | grep 'kubestellar-')
KUBECONFIG=~/.kube/config kubectl logs pod/$(kubectl get pods -n $ks_ns_edge_cluster1 \
    -o custom-columns=:metadata.name | grep 'kubestellar-') -n $ks_ns_edge_cluster1
```

and on the **ks-edge-cluster2** Kind cluster:

```shell
KUBECONFIG=~/.kube/config kubectl config use-context ks-edge-cluster2
ks_ns_edge_cluster2=$(KUBECONFIG=~/.kube/config kubectl get namespaces \
    -o custom-columns=:metadata.name | grep 'kubestellar-')
KUBECONFIG=~/.kube/config kubectl logs pod/$(kubectl get pods -n $ks_ns_edge_cluster2 \
    -o custom-columns=:metadata.name | grep 'kubestellar-') -n $ks_ns_edge_cluster2

```
<!--kubestellar-check-syncers-end-->