#!/bin/bash

echo Starting local system. This will take a while, take a coffee and relax

echo Checking if minikube is installed
minikube > /dev/null
if [ ! $? == 0 ]; then
	echo "minikube seems to not be installed or in path. Please install it first"
	exit 1
fi
echo Checking if minikube is running
RUNNING=$(minikube status| grep "apiserver: Running" | wc -l)
if [ $RUNNING != 1 ]; then
	echo "minikube is not running yet. trying to start it"
	minikube start && echo "minikube was started successfully"
fi
if [ ! $? == 0 ]; then
	echo "minikube could not be started"
	exit 1
fi

echo "everything seems fine. Starting deployment now :)"

echo "Deploying secret store csi driver"
helm repo add secrets-store-csi-driver https://kubernetes-sigs.github.io/secrets-store-csi-driver/charts
helm upgrade --install -n kube-system --set syncSecret.enabled=true csi-secrets-store secrets-store-csi-driver/secrets-store-csi-driver

echo "Building custom azure secret provider"
CDIR=$PWD
echo "getting docker context of kubernetes"
eval $(minikube -p minikube docker-env)

echo "building csi docker image"
docker build . -t csi-azure-fake-provider
cd $CDIR

echo "deploying fake secret store provider"
helm upgrade --install -n csi-azure-fake-provider --create-namespace csi-azure-fake-provider ./helm/

sleep 10

echo "Check if provider is running"
RUNNING=$(kubectl get pods -n csi-azure-fake-provider | grep Running)
if [ -z "$RUNNING"  ]; then
	echo "provider is not running. Please check logs"
	exit 1
fi

echo "Creating test secret value"
kubectl patch -n csi-azure-fake-provider cm/csi-azure-fake-provider-values --patch '{"data":{"test_secret": "test"}}'

echo "Deploying test secret store"
kubectl apply -f ./example-secrets.yml

echo "Deploying test pod"
kubectl apply -f ./example-pod.yml

echo "Waiting 20 seconds for pod to start and check logs"
sleep 20

POD=$(kubectl get pods -n default | grep hello | grep Running | awk '{print $1}')
if [ -z "$POD"  ]; then
	echo "pod is not running. Please check logs"
	exit 1
fi

echo "pod is running. Checking if secret was mounted correctly"
SECRET=$(kubectl exec -it hello -- sh -c 'echo $TEST_SECRET' | tr -d '\r')
if [ "$SECRET" != "test" ]; then
	echo "secret was not mounted correctly. Please check logs"
	exit 1
fi