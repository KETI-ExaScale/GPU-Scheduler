#!/usr/bin/env bash
dir=$( pwd )

#$1 create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then   
    echo kubectl delete -f $dir/../deployments/gpu-scheduler-polocy.yaml
    kubectl delete -f $dir/../deployments/gpu-scheduler-polocy.yaml
else
    echo kubectl apply -f $dir/../deployments/gpu-scheduler-polocy.yaml
    kubectl apply -f $dir/../deployments/gpu-scheduler-polocy.yaml
fi