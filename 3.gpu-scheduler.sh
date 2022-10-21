#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then 
    echo kubectl delete -f deployments/keti-gpu-scheduler.yaml
    kubectl delete -f deployments/keti-gpu-scheduler.yaml
else  
    echo kubectl create -f deployments/keti-gpu-scheduler.yaml
    kubectl create -f deployments/keti-gpu-scheduler.yaml
fi
