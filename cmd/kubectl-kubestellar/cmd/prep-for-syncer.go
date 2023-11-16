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

// "kubectl kubestellar prep-for-syncer" command
// Purpose: For the given SyncTarget, (a) prepare the corresponding
// mailbox workspace for the syncer and (b) output the YAML that needs
// to be created in the edge cluster to install the syncer there.

package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Create the Cobra sub-command for 'kubectl kubestellar prep-for-syncer'
func newPrepForSyncer(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make prep-for-syncer command
	cmdPrepForSyncer := &cobra.Command{
		Use:     "prep-for-syncer --output <FILENAME> --syncer-image <CONTAINER_IMG> --imw <IMW_NAME> --espw <ESPW_NAME>",
		Aliases: []string{"pfs"},
		Short:   "Prepare mailbox workspace, output YAML for edge cluster",
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := prepForSyncer(cmd, cliOpts, args, false)
			return err
		},
	}

	// Add required flag for output filename (--output or -o)
	cmdPrepForSyncer.Flags().StringVarP(&fname, "output", "o", "", "Output path/filename")
	cmdPrepForSyncer.MarkFlagRequired("output")
	cmdPrepForSyncer.MarkFlagFilename("output")
	return cmdPrepForSyncer
}

func init() {
	// Get config flags with default values.
	// Passing "true" will "use persistent client config, rest mapper,
	// discovery client, and propagate them to the places that need them,
	// rather than instantiating them multiple times."
	cliOpts := genericclioptions.NewConfigFlags(true)
	// Make a new flag set named prepForSyncer
	fs := pflag.NewFlagSet("prepForSyncer", pflag.ExitOnError)
	// Add cliOpts flags to fs (flow from syntax is confusing, goes -->)
	cliOpts.AddFlags(fs)
	// Add logging flags to fs
	fs.AddGoFlagSet(flag.CommandLine)
	// Add flags to our command; make these persistent (available to this
	// command and all sub-commands)
	rootCmd.PersistentFlags().AddFlagSet(fs)

	// Add sub-commands
	rootCmd.AddCommand(newPrepForSyncer(cliOpts))
}


func prepForSyncer(cmdGetKubeconfig *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string, isInternal bool) error {
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Print all flags and their values if verbosity level is at least 1
	cmdGetKubeconfig.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	// Get client config from flags
	config, err := cliOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get config from flags")
		return err
	}

	// Create client-go instance from config
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	return nil
}
