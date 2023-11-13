#!/usr/bin/env bash
dir=$( pwd )

#$1 create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then  
    echo kubectl delete -f $dir/../deployments/onegpupod.yaml
    kubectl delete -f $dir/../deployments/onegpupod.yaml
else 
    echo kubectl create -f $dir/../deployments/onegpupod.yaml
    kubectl create -f $dir/../deployments/onegpupod.yaml
fi
