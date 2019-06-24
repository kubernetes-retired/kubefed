# This Dockerfile represents a multistage build. The stages, respectively:
#
# 1. build kubefed binaries
# 2. copy binaries

# build stage 1: build kubefed binaries
FROM openshift/origin-release:golang-1.12 as builder
RUN yum update -y
RUN yum install -y make git

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

COPY . /go/src/github.com/openshift/kubefed/

WORKDIR /go/src/github.com/openshift/kubefed

RUN find . -name "*.go" -exec sed -i -r "s/sigs.k8s.io\/kubefed/github.com\/openshift\/kubefed/g"  {} \;

RUN sed -i "s/sigs.k8s.io/github.com\/openshift/g" Makefile 


RUN DOCKER_BUILD="/bin/sh -c " make hyperfed

# # build stage 2:
# FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base

# ENV USER_ID=1001

# # copy in binaries
# WORKDIR /root/
# COPY --from=builder /go/src/github.com/openshift/kubefed/bin/hyperfed-linux /root/hyperfed
# RUN ln -s hyperfed controller-manager && ln -s hyperfed kubefedctl &&  ln -s hyperfed webhook

# # user directive - this image does not require root
# USER ${USER_ID}

# ENTRYPOINT ["/root/controller-manager"]

# # apply labels to final image
# LABEL io.k8s.display-name="OpenShift KubeFed" \
#       io.k8s.description="This is a component that allows management of Kubernetes/OpenShift resources across multiple clusters" \
# maintainer="AOS Multicluster Team <aos-multicluster@redhat.com>"