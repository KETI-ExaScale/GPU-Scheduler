#!/bin/bash

POD_GPU_REQUEST=$1

PODNAME=""
echo "--------------------:: Create GPU Pod ::--------------------"

if [ "$1" == "gpu2" ] || [ "$2" == "2" ]; then
    kubectl create -f deployments/keti-two-gpu-pod.yaml
else
    kubectl create -f deployments/keti-one-gpu-pod.yaml
fi

sleep 1

while [ -z $PODNAME ]
do
    PODNAME=`kubectl get po -o=name -A --field-selector=status.phase=Running | grep gpu-scheduler` #pod/gpu-scheduler-d5bc65867-dkjnw
    PODNAME="${PODNAME:4}"
done

echo 
echo "--------------------:: KETI GPU Scheduler Log ::--------------------"
echo "--------------------------------------------------------------------"
echo

kubectl logs $PODNAME -n gpu -f