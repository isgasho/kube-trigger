## Examples

### RollingUpdate

Create CRD and operator

```
kubectl create -f ../deploy/crds/app_v1alpha1_triggerrule_crd.yaml
kubectl create -f operator.yaml
```

Create rolling update example

```
kubectl create -f ./rollingupdate/manifest.yaml
```

This example have a busybox app, which runs `tail -f /app/config/data /app/secret/data` to print content of configmap and secret.

Check pod logs 

```
kubectl -n example-rollingupdate logs <busybox podname>
```

Try to update configmap by edit it and you should observe that a new pod is created. Check logs to verify the update.