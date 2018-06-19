FROM ubuntu:14.04
RUN apt-get update
RUN apt-get install -y ca-certificates
ADD build/temp/apiserver .
ADD build/temp/controller-manager .

