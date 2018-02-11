# Sample container

This directory contains a `Dockerfile` that describes an image that runs a
`clusterregistry` aggregated to a standalone `kube-apiserver`, both of which
store data in an `etcd` instance. It also runs a `slackcontroller` that watches
for cluster addition and removal. The image does not set up any security
features, and should not serve as a model for a production deployment, but as an
example of all of the necessary component pieces working together in the
simplest possible fashion.

To build, from this directory, run:

```sh
bazel build //cmd/clusterregistry //examples/slackcontroller
cp "$(bazel info bazel-bin)/cmd/clusterregistry/clusterregistry" \
  "$(bazel info bazel-bin)/examples/slackcontroller/slackcontroller" \
  ./contents
docker build . -t crexample:latest
```

To run:

```sh
docker run -p 8080 crexample:latest <slack_incoming_webhook_url>
```

This will print out several messages as the various components being run are
started, but it should stabilize after several seconds and only be printing
messages about the `OpenAPI AggregationController` every minute. Run `docker ps`
to figure out the port that Docker has exposed, and access the API server
running in the container at `localhost:<port>`. For example,

```sh
$ kubectl get clusters -s localhost:<port>
No resources found
```
