#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then   
    echo kubectl delete -f deployments/keti-gpu-scheduler-concat.yaml
    kubectl delete -f deployments/keti-gpu-scheduler-concat.yaml
else
    echo kubectl create -f deployments/keti-gpu-scheduler-concat.yaml
    kubectl create -f deployments/keti-gpu-scheduler-concat.yaml
fi