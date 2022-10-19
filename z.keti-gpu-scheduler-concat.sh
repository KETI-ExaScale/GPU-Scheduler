#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "create" ] || [ "$1" == "c" ]; then
    echo kubectl create -f deployments/gpu-scheduler-concat.yaml
    kubectl create -f deployments/gpu-scheduler-concat.yaml
else
    echo kubectl delete -f deployments/gpu-scheduler-concat.yaml
    kubectl delete -f deployments/gpu-scheduler-concat.yaml
fi