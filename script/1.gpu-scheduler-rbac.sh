#!/usr/bin/env bash
dir=$( pwd )

#$1 create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then    
    echo kubectl delete -f $dir/../deployments/gpu-scheduler-role-binding.yaml
    echo kubectl delete -f $dir/../deployments/gpu-scheduler-service-account.yaml
    kubectl delete -f $dir/../deployments/gpu-scheduler-role-binding.yaml
    kubectl delete -f $dir/../deployments/gpu-scheduler-service-account.yaml
else
    echo kubectl create -f $dir/../deployments/gpu-scheduler-role-binding.yaml
    echo kubectl create -f $dir/../deployments/gpu-scheduler-service-account.yaml
    kubectl create -f $dir/../deployments/gpu-scheduler-role-binding.yaml
    kubectl create -f $dir/../deployments/gpu-scheduler-service-account.yaml
fi
