#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "create" ] || [ "$1" == "c" ]; then
    echo kubectl apply -f deployments/gpu-scheduler-configmap.yaml
    kubectl apply -f deployments/gpu-scheduler-configmap.yaml
else
    echo kubectl delete -f deployments/gpu-scheduler-configmap.yaml
    kubectl delete -f deployments/gpu-scheduler-configmap.yaml
fi