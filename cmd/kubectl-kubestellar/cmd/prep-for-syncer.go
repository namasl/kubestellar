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

	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	ksclientset "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned"
	plugin "github.com/kubestellar/kubestellar/pkg/cliplugins/kubestellar/prep-for-syncer"
)

var espw string // ESPW name, given by --espw flag
var imw string // IMW name, given by --imw flag
var syncerImageFname string // filename for syncer image, given by --syncer-image flag

// Create the Cobra sub-command for 'kubectl kubestellar prep-for-syncer'
func newPrepForSyncer(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make prep-for-syncer command
	cmdPrepForSyncer := &cobra.Command{
		Use:     "prep-for-syncer <SYNC_TARGET_NAME> --syncer-image <CONTAINER_IMG> --imw <IMW_NAME> --espw <ESPW_NAME> [--output <FILENAME>]",
		Aliases: []string{"pfs"},
		Short:   "Prepare mailbox workspace, output YAML for edge cluster",
		Args:    cobra.ExactArgs(1),
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

	// Add flags
	cmdPrepForSyncer.Flags().StringVarP(&fname, "output", "o", "", "Output path/filename")
	cmdPrepForSyncer.MarkFlagFilename("output")
	cmdPrepForSyncer.Flags().StringVarP(&syncerImageFname, "syncer-image", "", "", "Syncer container image")
	cmdPrepForSyncer.MarkFlagRequired("syncer-image")
	cmdPrepForSyncer.MarkFlagFilename("syncer-image")
	cmdPrepForSyncer.Flags().StringVarP(&imw, "imw", "", "", "IMW name")
	cmdPrepForSyncer.MarkFlagRequired("imw")
	cmdPrepForSyncer.Flags().StringVarP(&espw, "espw", "", "", "ESPW name")
	cmdPrepForSyncer.MarkFlagRequired("espw")
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
	syncTargetName := args[0]

	// Print all flags and their values if verbosity level is at least 1
	cmdGetKubeconfig.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	// --output flag is optional, make output filename if not provided
	if fname == "" {
		fname = syncTargetName + "-syncer.yaml"
	}

	// Set context to root
	// We will append workspace names to the root server as needed
	configContext := "root"
	cliOpts.Context = &configContext

	// Get client config from flags
	config, err := cliOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get config from flags")
		return err
	}

	// Keep a copy of the root server URL
	rootHost := config.Host

	// Create client-go instance from config
	rootClient, err := kcpclientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Update host to work on objects within IMW
	config.Host = rootHost + ":" + imw
	logger.V(1).Info(fmt.Sprintf("Set host to %s for IMW", config.Host))

	// Create client-go instance from config
	imwClient, err := ksclientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Update host to work on objects within ESPW
	config.Host = rootHost + ":" + espw
	logger.V(1).Info(fmt.Sprintf("Set host to %s for ESPW", config.Host))

	// Create client-go instance from config
	espwClient, err := kcpclientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	fmt.Println(rootClient, espwClient)
	fmt.Println("----------------")

	// Get mailbox name from SyncTarget
	// use imwClient
	// KUBECONFIG=~/ks-core.kubeconfig kubectl get synctargets.edge.kubestellar.io ks-edge-cluster1 -o jsonpath="{.metadata.annotations['kcp\.io/cluster']}-mb-{.metadata.uid}"
	// GET https://debian:1119/clusters/root:imw1/apis/edge.kubestellar.io/v2alpha1/synctargets/ks-edge-cluster1

	plugin.GetMailboxName(imwClient, ctx, syncTargetName)

	// in ESPW, check for APIExport edge.kubestellar.io
	// KUBECONFIG=~/ks-core.kubeconfig kubectl get APIExport edge.kubestellar.io
	// GET https://debian:1119/clusters/root:espw/apis/apis.kcp.io/v1alpha1/apiexports/edge.kubestellar.io
	// if that fails, warn that this is not the edge service provider workspace

	// in root, get mailbox
	// try, wait 15 seconds, try again, wait 15 seconds, try one last time
	// KUBECONFIG=~/ks-core.kubeconfig kubectl get Workspace d53tneij4e1yah6z-mb-0073399f-e2d6-4b61-b684-dfea16ca5bfc
	// GET https://debian:1119/clusters/root/apis/tenancy.kcp.io/v1alpha1/workspaces/d53tneij4e1yah6z-mb-0073399f-e2d6-4b61-b684-dfea16ca5bfc

	// Now workspace exists, but is it ready, wait 5 seconds

	// Work in mailbox workspace (make another client)

	// Check for APIBinding bind-edge
	// KUBECONFIG=~/ks-core.kubeconfig kubectl get APIBinding bind-edge
	// GET https://debian:1119/clusters/root:d53tneij4e1yah6z-mb-0073399f-e2d6-4b61-b684-dfea16ca5bfc/apis/apis.kcp.io/v1alpha1/apibindings/bind-edge

	// APIBinding exists, but has it taken effect? sleep 10


	// kubectl-kubestellar-syncer_gen" "$stname" --syncer-image "$syncer_image" -o "$output"


	return nil
}
