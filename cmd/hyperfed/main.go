/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// A binary that can morph into all of the other kubefed binaries. You can
// also soft-link to it busybox style.
//
package main

import (
	"errors"
	goflag "flag"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	_ "sigs.k8s.io/controller-runtime/pkg/metrics" // for workqueue metrics registration

	genericapiserver "k8s.io/apiserver/pkg/server"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Load all client auth plugins for GCP, Azure, Openstack, etc
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	ctrlapp "sigs.k8s.io/kubefed/cmd/controller-manager/app"
	webhookapp "sigs.k8s.io/kubefed/cmd/webhook/app"
	"sigs.k8s.io/kubefed/pkg/kubefedctl"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	hyperfedCommand, allCommandFns := NewHyperFedCommand()

	flags := hyperfedCommand.Flags()
	local := goflag.NewFlagSet(os.Args[0], goflag.ExitOnError)
	klog.InitFlags(local)
	flags.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	flags.AddGoFlagSet(local)

	logs.InitLogs()
	defer logs.FlushLogs()

	basename := filepath.Base(os.Args[0])
	if err := commandFor(basename, hyperfedCommand, allCommandFns).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1) //nolint:gocritic
	}
}

func commandFor(basename string, defaultCommand *cobra.Command, commands []func() *cobra.Command) *cobra.Command {
	for _, commandFn := range commands {
		command := commandFn()
		if command.Name() == basename {
			return command
		}
		for _, alias := range command.Aliases {
			if alias == basename {
				return command
			}
		}
	}

	return defaultCommand
}

// NewHyperFedCommand is the entry point for hyperfed
func NewHyperFedCommand() (*cobra.Command, []func() *cobra.Command) {
	stopChan := genericapiserver.SetupSignalHandler()

	controller := func() *cobra.Command { return ctrlapp.NewControllerManagerCommand(stopChan) }
	kubefedctlCmd := func() *cobra.Command { return kubefedctl.NewKubeFedCtlCommand(os.Stdout) }
	webhookCmd := func() *cobra.Command { return webhookapp.NewWebhookCommand(stopChan) }

	commandFns := []func() *cobra.Command{
		controller,
		kubefedctlCmd,
		webhookCmd,
	}

	makeSymlinksFlag := false
	cmd := &cobra.Command{
		Use:   "hyperfed COMMAND",
		Short: "Combined binary for KubeFed",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 || !makeSymlinksFlag {
				if err := cmd.Help(); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err.Error())
				}
				os.Exit(1)
			}

			if err := makeSymlinks(os.Args[0], commandFns); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err.Error())
			}
		},
	}
	cmd.Flags().BoolVar(&makeSymlinksFlag, "make-symlinks", makeSymlinksFlag, "create a symlink for each server in current directory")

	// hide this flag from appearing in servers' usage output
	if err := cmd.Flags().MarkHidden("make-symlinks"); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
	}

	for i := range commandFns {
		cmd.AddCommand(commandFns[i]())
	}

	return cmd, commandFns
}

// makeSymlinks will create a symlink for each command in the local directory.
func makeSymlinks(targetName string, commandFns []func() *cobra.Command) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	var errs bool
	for _, commandFn := range commandFns {
		command := commandFn()
		link := path.Join(wd, command.Name())

		err := os.Symlink(targetName, link)
		if err != nil {
			errs = true
			fmt.Println(err)
		}
	}

	if errs {
		return errors.New("error creating one or more symlinks")
	}
	return nil
}
