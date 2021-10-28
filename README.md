# scheduler

GPU Schedulerin kubernetes

## Build Scheduelr 

```
./1.build-scheduler.sh
```

## Run the Scheduler on Kubernetes

```
kubectl create -f deployments/gpu-scheduler.yaml
```
or
```
./2.update-scheduler.sh
```

deployment "gpu-scheduler" created


## Create a deployment

```
./3.create-gpupod.sh
```
or
```
./4.test.sh
```
