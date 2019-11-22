# kubectl http plugin

This plugin tests k8s ingress resource using interactive mode.

## How to use

```bash
cd kubectl-plugins/http
go build -o /usr/local/bin/kubectl-http .

# default namespace
kubectl http
# dev namespace
kubectl http -n dev
# specify ingress
kubectl http -n dev -i mock-ingress
```
