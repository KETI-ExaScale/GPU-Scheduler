#!/bin/bash

PODNAME=""
echo "--------------------:: Create GPU Pod ::--------------------"

while [ -z $PODNAME ]
do
    PODNAME=`kubectl get po -o=name -A --field-selector=status.phase=Running | grep gpu-scheduler` #pod/gpu-scheduler-d5bc65867-dkjnw
    PODNAME="${PODNAME:4}"
done
# echo $PODNAME #gpu-scheduler-d5bc65867-dkjnw
echo 
echo "--------------------:: KETI GPU Scheduler Log ::--------------------"
echo "--------------------------------------------------------------------"
sleep 1
kubectl logs $PODNAME -n gpu -f