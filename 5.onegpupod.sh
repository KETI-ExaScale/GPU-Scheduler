#!/usr/bin/env bash

#$1 create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then  
    echo kubectl delete -f deployments/onegpupod.yaml
    kubectl delete -f deployments/onegpupod.yaml
else 
    echo kubectl create -f deployments/onegpupod.yaml
    kubectl create -f deployments/onegpupod.yaml
fi
