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
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	ksclientset "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned"
	plugin "github.com/kubestellar/kubestellar/pkg/cliplugins/kubestellar/prep-for-syncer"
	syncergen "github.com/kubestellar/kubestellar/pkg/cliplugins/kubestellar/syncer-gen"
)

var espw string // ESPW name, given by --espw flag
var imw string // IMW name, given by --imw flag
var syncerImageName string // Syncer image name, given by --syncer-image flag
var fast bool // Indicates if we should skip waiting periods, given by --fast flag

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
	cmdPrepForSyncer.Flags().StringVarP(&syncerImageName, "syncer-image", "", "", "Syncer container image")
	cmdPrepForSyncer.MarkFlagRequired("syncer-image")
	cmdPrepForSyncer.MarkFlagFilename("syncer-image")
	cmdPrepForSyncer.Flags().StringVarP(&imw, "imw", "", "", "IMW name")
	cmdPrepForSyncer.MarkFlagRequired("imw")
	cmdPrepForSyncer.Flags().StringVarP(&espw, "espw", "", "", "ESPW name")
	cmdPrepForSyncer.MarkFlagRequired("espw")
	cmdPrepForSyncer.Flags().BoolVar(&fast, "fast", false, "Skip waiting periods")
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


	// Get mailbox name from named SyncTarget
	mbName, err := plugin.GetMailboxName(imwClient, ctx, syncTargetName)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem getting mailbox name for SyncTarget %s", syncTargetName))
		return err
	}
	logger.Info(fmt.Sprintf("Mailbox name is %s for SyncTarget %s", mbName, syncTargetName))

	// Verify that APIExport edge.kubestellar.io exists in ESPW
	exists, err := plugin.CheckAPIExportExists(espwClient, ctx, "edge.kubestellar.io")
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem checking for APIExport edge.kubestellar.io in ESPW %s", espw))
		return err
	}
	if !exists {
		logger.Error(err, fmt.Sprintf("APIExport edge.kubestellar.io does not exist in ESPW %s; is this the right workspace?", espw))
		return err
	}
	logger.Info(fmt.Sprintf("Found APIExport edge.kubestellar.io in ESPW %s", espw))

	// Verify we can get the mailbox workspace
	exists, err = plugin.CheckMailboxExists(rootClient, ctx, mbName)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem checking for mailbox %s", mbName))
		return err
	}
	if !exists {
		logger.Error(err, fmt.Sprintf("Could not find mailbox %s; is the mailbox controller running?", mbName))
		return err
	}
	logger.Info(fmt.Sprintf("Found mailbox %s", mbName))

	if !fast {
		// Wait 5 seconds after finding mailbox to give some buffer in hope it is ready
		logger.Info("Wait 5 seconds")
		time.Sleep(time.Second * 5)
	}

	// Set host to work on objects within mailbox workspace
	config.Host = rootHost + ":" + mbName
	logger.V(1).Info(fmt.Sprintf("Set host to %s for mailbox workspace", config.Host))

	// Create client-go instance from config
	mbClient, err := kcpclientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Check for APIBinding bind-edge in mailbox workspace
	exists, err = plugin.CheckAPIBindingExists(mbClient, ctx, "bind-edge")
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem checking for APIBinding bind-edge in mailbox workspace %s", mbName))
		return err
	}
	if !exists {
		logger.Error(err, fmt.Sprintf("APIBinding bind-edge does not exist in mailbox workspace %s; is this the right workspace?", mbName))
		return err
	}
	logger.Info(fmt.Sprintf("Found APIBinding bind-edge in mailbox workspace %s", mbName))

	if !fast {
		// Wait 10 seconds after finding APIBinding to give some buffer to ensure it has taken effect
		logger.Info("Wait 10 seconds")
		time.Sleep(time.Second * 10)
	}


	// Run syncer-gen to generate YAML manifest
	syncerGenOptions := syncergen.NewEdgeSyncOptions(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})

	syncerGenOptions.SyncerImage = syncerImageName
	syncerGenOptions.OutputFile = fname

	err = syncerGenOptions.Complete([]string{syncTargetName})
	if err != nil {
		logger.Error(err, "Problem with syncer-gen")
		return err
	}
	err = syncerGenOptions.Validate()
	if err != nil {
		logger.Error(err, "Problem with syncer-gen")
		return err
	}
	err = syncerGenOptions.Run(ctx)
	if err != nil {
		logger.Error(err, "Problem with syncer-gen")
		return err
	}

	return nil
}
