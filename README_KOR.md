# GPU Scheduler
**Contents**
- [1.GPU-Scheduler 소개](#introduction-of-GPU-Scheduler)
- [2.구성환경](#environment)
- [3.설치순서](#install-step)
- [4.모듈구성](#module-architecture)
----
## 1.GPU-Scheduler 소개 
쿠버네티스 기반 GPU 애플리케이션 컨테이너의 스케줄링을 돕는 GPU 자원 배치 스케줄러<br>
컨테이너간 GPU Sharing을 구현하고 GPU의 MPS동작을 지원한다.
#### 제공기능
- 동일 노드 내 GPU 자원 공유
- GPU-Metric-Collector의 자원 모니터링 기반 스케줄링
#### 필요모듈
- *[GPU-Metric-Collector](https://github.com/KETI-ExaScale/GPU-Metric-Collector)*
- *[GPU-Device-Plugin](https://github.com/KETI-ExaScale/GPU-Device-Plugin)*
- *[InfluxDB](https://github.com/KETI-ExaScale/InfluxDB)*
---
## 2.구성환경
모듈 설치는 다음과 같은 환경에서 구성되어야 한다.<br>
GPU Server/Node에는 GPU 컨테이너 실행을 위한 Nvidia-Docker 설치가 필요하다.<br>
- Kubernetes 
- Docker / Nvidia-Docker
클러스터 내에 'gpu' 네임스페이스가 있어야 한다. 다음과 같이 생성한다.<br>
'gpu' 네임스페이스에 스케줄러와 관련한 모듈이 설치된다.
```
# kubectl create namespace gpu
```
---
## 3.설치순서
### (1) host 이름 설정
Server/Node의 host이름을 설정하고 hosts 파일을 수정한다.
```
# hostnamectl set-hostname {your-hostname}
```
```
# vi /etc/hosts
```
### (2) Cluster rbac 설정
GPU-Scheduler 모듈에 권한부여를 위한 {role_binding.yaml service_account.yaml} 배포
```
[rbac yaml 배포]
# kubectl create -f deployments/rbac/role_binding.yaml
# kubectl create -f deployments/rbac/service_account.yaml
or
[rbac 쉘스크립트 실행]
# ./1.rbac-gpu-scheduler.sh
```
```
[root@master ~]# kubectl get serviceaccount -A
NAMESPACE         NAME                                 SECRETS   AGE
gpu               default                              1         19h
gpu               keti-gpu-device-plugin               1         21m
gpu               scheduler                            1         19h
```
### (3) GPU-Scheduler 정책 설정
GPU-Scheduler 정책 설정을 위한 {gpu-scheduler-configmap.yaml} 배포
```
[configmap yaml 배포]
# kubectl apply -f deployments/gpu-scheduler-configmap.yaml
or
[configmap 쉘스크립트 실행]
# ./2.polocy-config-gpu-scheduler.sh
```
```
[root@master ~]# kubectl get configmap -A
NAMESPACE         NAME                                 DATA   AGE
gpu               gpu-scheduler-configmap              2      19h
```
### (4) GPU-Scheduler 모듈 배포
클러스터 롤바인딩, 스케줄러 정책 설정 후 스케줄러 모듈 정상 배포 가능
GPU Scheduler 정상 작동을 위해 다음 3개의 모듈이 동작하고 있어야 함.  
*[GPU-Metric-Collector](https://github.com/KETI-ExaScale/GPU-Metric-Collector), [GPU-Device-Plugin](https://github.com/KETI-ExaScale/GPU-Device-Plugin), [InfluxDB](https://github.com/KETI-ExaScale/InfluxDB)*
```
[configmap yaml 배포]
# kubectl apply -f deployments/gpu-scheduler-configmap.yaml
or
[configmap 쉘스크립트 실행]
# ./2.polocy-config-gpu-scheduler.sh
```
최종적으로 다음과 같은 모듈 4개가 Running 상태일 때 정상 스케줄링 가능
```
[root@master ~]# kubectl get po -A
NAMESPACE     NAME                                  READY   STATUS      RESTARTS      AGE
gpu           influxdb-0                            1/1     Running     1 (19h ago)   19h
gpu           keti-gpu-device-plugin-h59db          2/2     Running     0             3m41s
gpu           keti-gpu-metric-collector-5q5hz       1/1     Running     0             22m
gpu           keti-gpu-scheduler-74c68ff868-6jmqr   1/1     Running     1 (42m ago)   18h
```
### (5) GPU 응용 컨테이너 배포
GPU 응용 애플리케이션의 배포
#### Pod yaml 필수요소
+ 해당 커스텀 GPU-Scheduler 사용을 위한 schedulerName 설정
```
    spec:
      schedulerName: gpu-scheduler # GPU-Scheduler 사용을 위한 스케줄러 지정
```
+ keti.com/mpsgpu 리소스 요청을 통한 GPU 개수 지정
```
        resources:
            limits:
              keti.com/mpsgpu: 1 # GPU개수 요청 설정
```
+ GPU MPS 지원
```
        volumeMounts: # GPU MPS 지원
            - name: nvidia-mps
              mountPath: /tmp/nvidia-mps 
      volumes: # GPU MPS 지원
        - name: nvidia-mps
          hostPath:
            path: /tmp/nvidia-mps
```
#### GPU-Scheduler를 통해 스케줄링되는 Pod yaml 예시
```
apiVersion: batch/v1
kind: Job
metadata:
  name: nbody-benchmark-mps-test # 파드 이름 지정
  namespace: userpod # 파드 네임스페이스 지정
spec:
  template:
    spec:
      hostIPC: true
      schedulerName: gpu-scheduler # GPU-Scheduler 사용을 위한 스케줄러 지정
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
              keti.com/mpsgpu: 1 # GPU개수 요청 설정
              cpu: "250m"
              memory: "6400Mi"
            requests:
              cpu: "250m"
              memory: "6400Mi"
          volumeMounts: # GPU MPS 지원
            - name: nvidia-mps
              mountPath: /tmp/nvidia-mps 
      volumes: # GPU MPS 지원
        - name: nvidia-mps
          hostPath:
            path: /tmp/nvidia-mps
      restartPolicy: Never
```
#### GPU-Scheduler를 통한 GPU 응용 배포
```
[test pod yaml 배포]
# kubectl apply -f deployments/testgpupod.yaml
or
[pod create 쉘스크립트 실행]
# ./4.test-gpupod.sh
```