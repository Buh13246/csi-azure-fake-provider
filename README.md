# csi-azure-fake-provider
This csi azure fake provider is created to deploy helm charts with secrets from azure keyvault in minikube for local testing

STILL UNDER DEVELOPMENT BUT WORKS (KINDA)

# Quick Install
There is a shellscript called install.sh
Which will create a minikube cluster, install everything needed to work and tests everything with a test container (busybox)

# Manual Install
First install (if not present yet) the kubernetes secrets-store-csi-driver:
```
helm upgrade --install -n kube-system --set syncSecret.enabled=true csi-secrets-store secrets-store-csi-driver/secrets-store-csi-driver
```

Next install helm chart for the fake provider
```
helm upgrade --install -n csi-azure-fake-provider --create-namespace csi-azure-fake-provider ./helm/
```

This created a namespace "csi-azure-fake-provider" where the fake azure provider lives. There it created a configmap called "csi-azure-fake-provider-values". This is where you can insert your values which will than be used as secrets.
```
kubectl edit -n csi-azure-fake-provider cm/csi-azure-fake-provider-values
```