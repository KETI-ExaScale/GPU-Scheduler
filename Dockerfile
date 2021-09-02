FROM ubuntu:16.04
ADD gpu-scheduler /gpu-scheduler
ENTRYPOINT ["/gpu-scheduler"]
