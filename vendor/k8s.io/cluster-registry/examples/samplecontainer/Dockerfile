FROM ubuntu:latest

ARG ETCD_VERSION=3.2.9
ARG KUBE_VERSION=1.8.3

RUN mkdir -p /sample

WORKDIR /sample

RUN apt-get -qq update && apt-get -y install curl iproute2
RUN curl -Lo kubernetes.tar.gz https://dl.k8s.io/v$KUBE_VERSION/kubernetes-server-linux-amd64.tar.gz
RUN curl -Lo etcd.tar.gz https://github.com/coreos/etcd/releases/download/v$ETCD_VERSION/etcd-v$ETCD_VERSION-linux-amd64.tar.gz
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v$KUBE_VERSION/bin/linux/amd64/kubectl

RUN chmod +x kubectl
RUN tar xzf etcd.tar.gz
RUN tar xzf kubernetes.tar.gz

ADD ["contents/*", "/sample/"]

ENTRYPOINT ["./run.sh"]

EXPOSE 8080
