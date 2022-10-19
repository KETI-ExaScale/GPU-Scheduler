# GPU Scheduler
**Contents**
- [1.Introduction-of-GPU-Scheduler](#1-introduction-of-GPU-Scheduler)
- [2.Environment](#2-environment)
- [3.Installation](#3-installation)
----
## 1.Introduction-of-GPU-Scheduler
GPU resource scheduler that helps schedule kubernetes-based GPU application containers.<br>
It implements GPU sharing between containers and supports the MPS operation of GPUs.
#### Main Function
- GPU resource sharing within one node.
- GPU-Metric-Collector's Resource monitoring-based scheduling.
#### required module
- *[GPU-Metric-Collector](https://github.com/KETI-ExaScale/GPU-Metric-Collector)*
- *[GPU-Device-Plugin](https://github.com/KETI-ExaScale/GPU-Device-Plugin)*
- *[KETI-Cluster-Manager](https://github.com/KETI-ExaScale/KETI-Cluster-Manager)*
- *[InfluxDB](https://github.com/KETI-ExaScale/InfluxDB)*
---
## 2.Environment
Module installation proceeds in the following environment..<br>
GPU Server/Node requires Nvidia-Docker for GPU container execution.<br>
- Kubernetes 
- Docker / Nvidia-Docker
The cluster must have a 'gpu' namespace and is created as follows.<br>
A module related to the scheduler is installed in the 'gpu' namespace.
```
# kubectl create namespace gpu
```
---
## 3.Installation
### (1) Set the host name
Set the host name of the server/Node and modify the hosts file.
```
# hostnamectl set-hostname {your-hostname}
```
```
# vi /etc/hosts
```
### (2) Set Cluster rbac
Create for authorization to GPU-Scheduler
```
[create rbac yaml]
# kubectl create -f deployments/rbac/role_binding.yaml
# kubectl create -f deployments/rbac/service_account.yaml
or
[run rbac script]
# ./1.rbac-gpu-scheduler.sh
```
```
[root@master ~]# kubectl get serviceaccount -A
NAMESPACE         NAME                                 SECRETS   AGE
gpu               default                              1         19h
gpu               keti-gpu-device-plugin               1         21m
gpu               scheduler                            1         19h
```
### (3) Set GPU-Scheduler Policy
Create {gpu-scheduler-configmap.yaml} for setting policy to GPU-Scheduler
```
[create configmap yaml]
# kubectl apply -f deployments/gpu-scheduler-configmap.yaml
or
[run configmap script]
# ./2.polocy-config-gpu-scheduler.sh
```
```
[root@master ~]# kubectl get configmap -A
NAMESPACE         NAME                                 DATA   AGE
gpu               gpu-scheduler-configmap              2      19h
```
### (4) Create GPU-Scheduler
Successful deployment of scheduler modules is possible after cluster roll binding and scheduler policy setting.
The following three modules must be operating for GPU Scheduler Successful operation.<br>
*[GPU-Metric-Collector](https://github.com/KETI-ExaScale/GPU-Metric-Collector), [GPU-Device-Plugin](https://github.com/KETI-ExaScale/GPU-Device-Plugin), [InfluxDB](https://github.com/KETI-ExaScale/InfluxDB)*
```
[create gpu-scheduler yaml]
# kubectl apply -f deployments/gpu-scheduler-configmap.yaml
or
[run gpu-scheduler script]
# ./2.polocy-config-gpu-scheduler.sh
```
Finally, successful scheduling is possible when the following 4 modules are running.
```
[root@master ~]# kubectl get po -A
NAMESPACE     NAME                                  READY   STATUS      RESTARTS      AGE
gpu           influxdb-0                            1/1     Running     1 (19h ago)   19h
gpu           keti-gpu-device-plugin-h59db          2/2     Running     0             3m41s
gpu           keti-gpu-metric-collector-5q5hz       1/1     Running     0             22m
gpu           keti-gpu-scheduler-74c68ff868-6jmqr   1/1     Running     1 (42m ago)   18h
```
### (5) Create GPU Pod
#### Essential elements of GPU Pod yaml
+ Set schedulerName for using suctom GPU-Scheduler
```
    spec:
      schedulerName: gpu-scheduler # GPU-Scheduler Name
```
+ Set GPU count requiring keti.com/mpsgpu resource
```
        resources:
            limits:
              keti.com/mpsgpu: 1 # GPU Count
```
+ Using GPU MPS for performance improvement
```
        volumeMounts: # GPU MPS
            - name: nvidia-mps
              mountPath: /tmp/nvidia-mps 
      volumes: # GPU MPS
        - name: nvidia-mps
          hostPath:
            path: /tmp/nvidia-mps
```
+ IPC Namespace Sharing Settings for Host Containers
```
hostIPC: true
```
#### GPU Pod Yaml Example
```
apiVersion: batch/v1
kind: Job
metadata:
  name: nbody-benchmark-mps-test # Pod Name
  namespace: userpod # Pod Namespace
spec:
  template:
    spec:
      hostIPC: true
      schedulerName: gpu-scheduler # Using GPU-Scheduler
      containers:
        - image: seedjeffwan/nbody:cuda-10.1
          name: nbody1
          args:
            - nbody
            - -benchmark
            - -numdevices=1
            - -numbodies=1982000
          resources:
            limits:
              keti.com/mpsgpu: 1 # Set GPU Count
              cpu: "250m"
              memory: "6400Mi"
            requests:
              cpu: "250m"
              memory: "6400Mi"
          volumeMounts: # GPU MPS
            - name: nvidia-mps
              mountPath: /tmp/nvidia-mps 
      volumes: # GPU MPS
        - name: nvidia-mps
          hostPath:
            path: /tmp/nvidia-mps
      restartPolicy: Never
```
#### GPU Pod Creatrion by GPU-Scheduler
```
[test pod yaml creat]
# kubectl apply -f deployments/testgpupod.yaml
or
[run pod create script]
# ./4.test-gpupod.sh
```
#### Pod creation result
```
[root@master ~]# kubectl get po -A
NAMESPACE     NAME                                  READY   STATUS      RESTARTS      AGE
gpu           influxdb-0                            1/1     Running     1 (19h ago)   19h
gpu           keti-gpu-device-plugin-h59db          2/2     Running     0             3m41s
gpu           keti-gpu-metric-collector-5q5hz       1/1     Running     0             22m
gpu           keti-gpu-scheduler-74c68ff868-6jmqr   1/1     Running     1 (42m ago)   18h
userpod       testpod                               0/1     Completed   0             18m
```
