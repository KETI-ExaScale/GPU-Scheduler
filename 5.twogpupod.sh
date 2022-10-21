#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then 
    echo kubectl delete -f deployments/twogpupod.yaml
    kubectl delete -f deployments/twogpupod.yaml
else  
    echo kubectl create -f deployments/twogpupod.yaml
    kubectl create -f deployments/twogpupod.yaml
fi
