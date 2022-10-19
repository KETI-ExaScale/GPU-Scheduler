#!/usr/bin/env bash

arg=$1 # create/c or delete/d

if [ "$1" == "create" ] || [ "$1" == "c" ]; then
    echo kubectl create -f deployments/rbac/role_binding.yaml
    echo kubectl create -f deployments/rbac/service_account.yaml
    kubectl create -f deployments/rbac/role_binding.yaml
    kubectl create -f deployments/rbac/service_account.yaml
else
    echo kubectl delete -f deployments/rbac/role_binding.yaml
    echo kubectl delete -f deployments/rbac/service_account.yaml
    kubectl delete -f deployments/rbac/role_binding.yaml
    kubectl delete -f deployments/rbac/service_account.yaml
fi
