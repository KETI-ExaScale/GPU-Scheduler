#!/bin/bash

while [ -z $PODNAME ]
do
    PODNAME=`kubectl get po -o=name -A --field-selector=status.phase=Running | grep gpu-scheduler`
    PODNAME="${PODNAME:4}"
done

kubectl logs $PODNAME -n gpu -f