#!/usr/bin/env bash
dir=$( pwd )

#$1 create/c or delete/d

if [ "$1" == "delete" ] || [ "$1" == "d" ]; then  
    echo kubectl delete -f $dir/../deployments/userpod-nbody-benchmark1.yaml
    kubectl delete -f $dir/../deployments/userpod-nbody-benchmark1.yaml
else 
    echo kubectl create -f $dir/../deployments/userpod-nbody-benchmark1.yaml
    kubectl create -f $dir/../deployments/userpod-nbody-benchmark1.yaml
fi