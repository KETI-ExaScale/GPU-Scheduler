#!/bin/bash

PODNAMES=""
while [ -z $PODNAMES ]
do
    PODNAMES=`kubectl get po  -A -o=name -o wide  --field-selector=status.phase=Running | grep flannel | grep gpu-server`
done

LIST=($PODNAMES)
echo ${LIST[1]}

# for PODNAME in $PODNAMES
# do
#     # PODNAME="${PODNAME:4}"
# 	echo $PODNAME
#     i
# done

# echo 
# sleep 1
# echo "--------------------:: KETI GPU Metric Collector Log ::--------------------"
# echo "---------------------------------------------------------------------------"
# sleep 1
# kubectl logs $PODNAME -n gpu --tail=19