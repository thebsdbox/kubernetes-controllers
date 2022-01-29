# vcluster controller

## Proof of concept Controller for vClusters from Loft

This will provision a single vcluster from a manifest once the CRDs are installed and teh controller is up and running.

### Sample Manifest

```
apiVersion: vcluster.fnnrn.me/v1
kind: Cluster
metadata:
  name: cluster-sample
spec:
  name: example
  namespace: default
  cidr: "10.96.0.0/12"
  version: "0.5.3"
```

