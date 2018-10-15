#!/bin/bash -e

OLD_POD=$(kubectl get po -l app=controlplane-pipeline --sort-by=.metadata.creationTimestamp -o=name|tail -n1 )
echo Old POD name: $OLD_POD

time make debug-docker
echo "DELETE the old POD: $OLD_POD"
kubectl delete $OLD_POD

sleep 5

echo "Force DELETE the old POD: $OLD_POD"
kubectl delete $OLD_POD --grace-period=0 --force

NEW_POD=$(kubectl get po -l app=controlplane-pipeline --sort-by=.metadata.creationTimestamp -o=name|tail -n1 )
NEW_POD_STATUS=""
echo "New POD Name: $NEW_POD"

while [[ "$NEW_POD_STATUS" != "Running" ]] ; do
    sleep 2
    NEW_POD_STATUS=$(kubectl get $NEW_POD -o json | jq -j '.status.phase'| grep -v null)
    echo "New POD Status: $NEW_POD_STATUS"
done
osascript -e 'display notification "Proxy activated" with title "New Build Deployed"'

echo "Start Proxy"
kubectl port-forward $NEW_POD 40000
