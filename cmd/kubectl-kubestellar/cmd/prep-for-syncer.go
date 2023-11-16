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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	plugin "github.com/kubestellar/kubestellar/pkg/cliplugins/kubestellar/get-kubeconfig"
)

var imw string // IMW name
var espw string // ESPW name
var syncerImage string // Filename for syncer image

// Create the Cobra sub-command for 'kubectl kubestellar prep-for-syncer'
func newPrepForSyncer(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make prep-for-syncer command
	cmdGetExternalKubeconfig := &cobra.Command{
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
			err := getKubeconfig(cmd, cliOpts, args, false)
			return err
		},
	}

	// Add required flag for output filename (--output or -o)
	cmdGetExternalKubeconfig.Flags().StringVarP(&fname, "output", "o", "", "Output path/filename")
	cmdGetExternalKubeconfig.MarkFlagRequired("output")
	return cmdGetExternalKubeconfig
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

// Get KubeStellar kubeconfig, and write to output file (filename given by fname
// variable, tied to --output flag).
func prepForSyncer(cmdGetKubeconfig *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string, isInternal bool) error {
	ctx := context.Background()
	logger := klog.FromContext(ctx)
	// Set context from KUBECONFIG to use in client
	configContext := ksContext
	cliOpts.Context = &configContext

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

	// Get name of KubeStellar server pod
	serverPodName, err := plugin.GetServerPodName(client, ctx, ksNamespace, ksSelector)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem finding server pod in namespace %s with selector %s", ksNamespace, ksSelector))
		return err
	}
	logger.Info(fmt.Sprintf("Found KubeStellar server pod %s", serverPodName))

	// Check if server pod is ready
	err = plugin.KsPodIsReady(client, config, ksNamespace, serverPodName, "init")
	if err != nil {
		logger.Error(err, fmt.Sprintf("KubeStellar init container in pod %s is not ready", serverPodName))
		return err
	}
	logger.Info(fmt.Sprintf("KubeStellar init container in pod %s is ready", serverPodName))

	// Get KubeSteallar Kubeconfig
	ksConfig, err := plugin.GetKSKubeconfig(client, ctx, ksNamespace, isInternal)
	if err != nil {
		logger.Error(err, "Problem obtaining kubeconfig")
		return err
	}
	logger.V(1).Info(fmt.Sprintf("kubeconfig: %s", string(ksConfig)))

	// Write to file
	err = os.WriteFile(fname, ksConfig, 0644)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem writing kubeconfig to output file %s", fname))
		return err
	}
	logger.Info(fmt.Sprintf("Wrote kubeconfig to file %s", fname))

	return nil
}
