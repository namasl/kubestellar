/*
Copyright 2023 The KubeStellar Authors.
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

// This is the sub-command for getting the KubeStellar kubectl configuration.
// "get-external-kubeconfig" is used when running externally to the cluster hosting Kubestellar.
// "get-internal-kubeconfig" is used when running inside the same cluster as Kubestellar.

package cmd

import (
    "context"
    "fmt"
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

var fname string // Filename/path for output configuration file

// Create the Cobra sub-command for 'kubectl kubestellar get-external-kubeconfig'
func newGetExternalKubeconfig(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make wds command
	cmdGetExternalKubeconfig := &cobra.Command{
		Use:     "get-external-kubeconfig",
		Aliases: []string{"gek"},
		Short:   "Get KubeStellar kubectl configuration when external to host cluster",
		Args:    cobra.ExactArgs(0),
		RunE:    func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := getKubeconfig(cmd, cliOpts, args, false)
			return err
		},
	}

	// Add required flag for output filename (--output or -o)
	cmdGetExternalKubeconfig.Flags().StringVarP(&fname, "output", "o", "", "Output path/filename")
	cmdGetExternalKubeconfig.MarkFlagRequired("output")
	return cmdGetExternalKubeconfig
}

// Create the Cobra sub-command for 'kubectl kubestellar get-internal-kubeconfig'
func newGetInternalKubeconfig(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make wds command
	cmdGetInternalKubeconfig := &cobra.Command{
		Use:     "get-internal-kubeconfig",
		Aliases: []string{"gek"},
		Short:   "Get KubeStellar kubectl configuration from inside same cluster",
		Args:    cobra.ExactArgs(0),
		RunE:    func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := getKubeconfig(cmd, cliOpts, args, true)
			return err
		},
	}

	// Add required flag for output filename (--output or -o)
	cmdGetInternalKubeconfig.Flags().StringVarP(&fname, "output", "o", "", "Output path/filename")
	cmdGetInternalKubeconfig.MarkFlagRequired("output")
	return cmdGetInternalKubeconfig
}

func init() {
	// Get config flags with default values.
	// Passing "true" will "use persistent client config, rest mapper,
	// discovery client, and propagate them to the places that need them,
	// rather than instantiating them multiple times."
	cliOpts := genericclioptions.NewConfigFlags(true)
	// Make a new flag set named getKubeconfig
	fs := pflag.NewFlagSet("getKubeconfig", pflag.ExitOnError)
	// Add cliOpts flags to fs (flow from syntax is confusing, goes -->)
	cliOpts.AddFlags(fs)
	// Add logging flags to fs
	fs.AddGoFlagSet(flag.CommandLine)
	// Add flags to our command; make these persistent (available to this
	// command and all sub-commands)
	rootCmd.PersistentFlags().AddFlagSet(fs)

	// Add sub-commands
	rootCmd.AddCommand(newGetExternalKubeconfig(cliOpts))
	rootCmd.AddCommand(newGetInternalKubeconfig(cliOpts))
}

func getKubeconfig(cmdGetExternalKubeconfig *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string, isInternal bool) error {
	ctx := context.Background()
	logger := klog.FromContext(ctx)

    fmt.Println("GEK!")
    logger.V(1).Info("222")
    return nil
}
