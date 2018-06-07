# Provides stamped variables from the Bazel workspace status. Use with the
# x_defs field of a go_binary rule.
def version_x_defs():
  stamp_pkgs = ["k8s.io/cluster-registry/pkg/version"]
  # It should match the list of vars set in hack/print-workspace-status.sh.
  stamp_vars = [
      "buildDate",
      "gitCommit",
      "gitTreeState",
      "semanticVersion",
  ]
  # Generate the cross-product.
  x_defs = {}
  for pkg in stamp_pkgs:
    for var in stamp_vars:
      x_defs["%s.%s" % (pkg, var)] = "{%s}" % var
  return x_defs
