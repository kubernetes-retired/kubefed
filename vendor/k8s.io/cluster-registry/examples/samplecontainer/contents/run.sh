#!/bin/bash

etcd-v3.2.9-linux-amd64/etcd &

kubernetes/server/bin/kube-apiserver \
  --insecure-bind-address 0.0.0.0 \
  --etcd-servers http://127.0.0.1:2379 \
  --service-cluster-ip-range=10.10.0.0/14 &

./clusterregistry \
  --secure-port 443 \
  --etcd-servers http://127.0.0.1:2379 &

while [[ $(curl http://localhost:8080/healthz 2>/dev/null) != "ok" ]]; do
  sleep 1
done

./kubectl create -f cr-service.yaml -s http://localhost:8080
./kubectl create -f cr-apiservice.yaml -s http://localhost:8080
./slackcontroller -kubeconfig ./config -slack-url "$@"
