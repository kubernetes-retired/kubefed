# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Rules for generating gocode at compile time."""

load("@bazel_skylib//lib:paths.bzl", "paths")
load("@io_bazel_rules_go//go:def.bzl", "GoArchive", "GoLibrary", "go_context", "go_rule")

def _compute_genrule_variables(srcs, outs):
    resolved_srcs = [src.path for src in srcs]
    resolved_outs = [out.path for out in outs]
    variables = {
        "SRCS": " ".join(resolved_srcs),
        "OUTS": " ".join(resolved_outs),
    }
    if len(resolved_srcs) == 1:
        variables["<"] = resolved_srcs[0]
    if len(resolved_outs) == 1:
        variables["@"] = resolved_outs[0]
    return variables

def _go_genrule_impl(ctx):
    go = go_context(ctx)

    transitive_libs = depset(transitive = [d[GoArchive].transitive for d in ctx.attr.go_deps])

    gopath = []
    for lib in transitive_libs.to_list():
        for srcfile in lib.srcs:
            target = ctx.actions.declare_file(paths.join(
                "gopath/src",
                lib.importpath,
                paths.basename(srcfile.path),
            ))

            args = ctx.actions.args()
            args.add(srcfile.path)
            args.add(target.path)

            ctx.actions.run(
                executable = "mv",
                arguments = [args],
                inputs = [srcfile],
                outputs = [target],
                mnemonic = "PrepareGopath",
            )

            gopath.append(target)

    srcs = [src for srcs in ctx.attr.srcs for src in srcs.files.to_list()]

    deps = depset(
        gopath + srcs,
        transitive =
            # tools
            [dep.files for dep in ctx.attr.tools] +
            # go toolchain
            [depset(go.sdk.libs + go.sdk.srcs + go.sdk.tools + [go.sdk.go])],
    )

    _, cmd, _ = ctx.resolve_command(
        command = ctx.attr.cmd,
        attribute = "cmd",
        expand_locations = True,
        make_variables = _compute_genrule_variables(
            srcs,
            ctx.outputs.outs,
        ),
        tools = ctx.attr.tools,
    )

    env = dict()
    env.update(ctx.configuration.default_shell_env)
    env.update(go.env)
    env.update({
        "PATH": ctx.configuration.host_path_separator.join(["/usr/local/bin", "/bin", "/usr/bin"]),
        "GOPATH": paths.join(ctx.bin_dir.path, paths.dirname(ctx.build_file_path), "gopath"),
        "GOROOT": paths.dirname(go.sdk.root_file.path),
    })

    ctx.actions.run_shell(
        inputs = deps,
        outputs = ctx.outputs.outs,
        env = env,
        command = cmd,
        progress_message = "%s %s" % (ctx.attr.message, ctx),
        mnemonic = "GoGenrule",
    )

# We have codegen procedures that depend on the "go/*" stdlib packages
# and thus depend on executing with a valid GOROOT. _go_genrule handles
# dependencies on the Go toolchain and environment variables; the
# macro go_genrule handles setting up GOPATH dependencies (using go_path).
go_genrule = go_rule(
    _go_genrule_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "tools": attr.label_list(
            cfg = "host",
            allow_files = True,
        ),
        "outs": attr.output_list(mandatory = True),
        "cmd": attr.string(mandatory = True),
        "go_deps": attr.label_list(providers = [
            GoLibrary,
            GoArchive,
        ]),
        "importpath": attr.string(),
        "message": attr.string(),
        "executable": attr.bool(default = False),
    },
    output_to_genfiles = True,
)
