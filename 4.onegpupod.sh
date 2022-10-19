#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "create" ] || [ "$1" == "c" ]; then
    echo kubectl create -f deployments/onegpupod.yaml
    kubectl create -f deployments/onegpupod.yaml
else
    echo kubectl delete -f deployments/onegpupod.yaml
    kubectl delete -f deployments/onegpupod.yaml
fi
