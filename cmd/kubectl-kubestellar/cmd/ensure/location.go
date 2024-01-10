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

// Sub-command for ensuring the existence and configuration a location in a WEC.
// The IMW is provided by the required --imw flag.
// The location name is provided as a required command-line argument.
// Labels in key=value pairs are provided as command-line arguments, for which
// we will ensure that these exist as labels in the Location and SyncTarget.

package ensure

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	clientset "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned"
	plugin "github.com/kubestellar/kubestellar/pkg/cliplugins/kubestellar/ensure"
)

// Create the Cobra sub-command for 'kubectl kubestellar ensure location'
func newCmdEnsureLocation(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make location command
	cmdLocation := &cobra.Command{
		Use:     "location <LOCATION_NAME> <LABEL_1 ...>",
		Aliases: []string{"loc"},
		Short:   "Ensure existence and configuration of an inventory listing for a WEC",
		// We actually require at least 2 arguments (location name and a label),
		// but more descriptive error messages will be provided by leaving this
		// set to 1.
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := ensureLocation(cmd, args, cliOpts)
			return err
		},
	}

	return cmdLocation
}

// Location name is the first command line argument.
// Labels to check are provided as additional arguments in key=value pairs,
// with at least one required.
func ensureLocation(cmdLocation *cobra.Command, args []string, cliOpts *genericclioptions.ConfigFlags) error {
	locationName := args[0]
	labels := args[1:]
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Print all flags and their values if verbosity level is at least 1
	cmdLocation.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	// Make sure user provided location name is valid
	err := plugin.CheckLocationName(locationName)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem with location name %s", locationName))
		return err
	}

	// Make sure user provided labels are valid
	err = plugin.CheckLabelArgs(labels)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem with label arguments %s", labels))
		return err
	}

	// Set context to root
	configContext := "root"
	cliOpts.Context = &configContext

	// Get client config from flags
	config, err := cliOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get config from flags")
		return err
	}

	// Create client-go instance from config
	client, err := clientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Check that SyncTarget exists and is configured, create/update if not
	// This function prints its own log messages, so no need to add any here.
	err = plugin.VerifyOrCreateSyncTarget(client, ctx, locationName, labels)
	if err != nil {
		return err
	}

	// Check if Location exists and is configured, create/update if not
	// This function prints its own log messages, so no need to add any here.
	err = plugin.VerifyOrCreateLocation(client, ctx, locationName, labels)
	if err != nil {
		return err
	}

	return nil
}
