# kubectl revision plugin

## How to use

```bash
cd kubectl-plugins/revisions
go build -o kubectl-revisions main.go

cp kubectl-revisions /usr/local/bin

kubectl revisions -d <deployment-name> -n <namespace>
```

```bash
Rev     Version CreatedAt               Image                           Status
1       230946  2019-11-02 14:26        kudohn/github-service:flux-v1   Terminated
2       240037  2019-11-02 14:30        kudohn/github-service:flux-v2   Ready
```
