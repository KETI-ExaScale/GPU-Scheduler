kubectl delete -f deployments/onegpupod.yaml
kubectl delete -f deployments/twogpupod.yaml
kubectl create -f deployments/onegpupod.yaml
kubectl create -f deployments/twogpupod.yaml