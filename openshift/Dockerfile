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

WORKDIR /go/src/sigs.k8s.io/kubefed

# copy in git info to support recording version in binaries
COPY .git/HEAD .git/HEAD
COPY .git/refs/heads/. .git/refs/heads
RUN mkdir -p .git/objects

COPY Makefile Makefile
COPY pkg pkg
COPY cmd cmd
COPY test test
COPY vendor vendor

RUN DOCKER_BUILD="/bin/sh -c " make hyperfed

# build stage 2:
FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base

ENV USER_ID=1001

# copy in binaries
WORKDIR /root/
COPY --from=builder /go/src/sigs.k8s.io/kubefed/bin/hyperfed-linux /root/hyperfed
RUN ln -s hyperfed controller-manager && ln -s hyperfed kubefedctl &&  ln -s hyperfed webhook

# user directive - this image does not require root
USER ${USER_ID}

ENTRYPOINT ["/root/controller-manager"]
CMD ["--install-crds=false"]

# apply labels to final image
LABEL io.k8s.display-name="OpenShift KubeFed" \
      io.k8s.description="This is a component that allows management of Kubernetes/OpenShift resources across multiple clusters" \
maintainer="AOS Multicluster Team <aos-multicluster@redhat.com>"
