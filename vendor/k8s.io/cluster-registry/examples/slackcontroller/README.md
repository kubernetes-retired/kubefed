# SlackController

A sample controller that demonstrates how to write a controller that targets the
cluster registry. This controller will post messages to a Slack channel when a
cluster is added to or removed from the registry.

## Quickstart

1.  Set up a cluster registry. You can use
    [crinit](/docs/userguide.md#deploying-a-cluster-registry) to do so.
1.  Run `bazel build //examples/slackcontroller` to build the controller.
1.  Create a [Slack incoming webhook](https://api.slack.com/incoming-webhooks)
    and get its URL to pass via the `-slack-url` flag on `slackcontroller`.
1.  Deploy the cluster registry into a cluster, making sure you pass the
    incoming webhook URL to its `-slack-url` flag.
