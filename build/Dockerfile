FROM ubuntu:16.04

ENV OPERATOR=/usr/local/bin/keti-gpu-scheduler \
    USER_UID=root \
    USER_NAME=discovery
COPY _output/bin/keti-gpu-scheduler ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/keti-gpu-scheduler"]

USER ${USER_UID}