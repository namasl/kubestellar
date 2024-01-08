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

	// kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	// clientset "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned"
	// plugin "github.com/kubestellar/kubestellar/pkg/cliplugins/kubestellar/ensure"
)

var imw string // IMW name, provided by --imw flag

// Create the Cobra sub-command for 'kubectl kubestellar ensure location'
func newCmdEnsureLocation(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make location command
	cmdLocation := &cobra.Command{
		Use:     "location --imw <IMW_NAME> <LOCATION_NAME> <LABEL_1 ...>",
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

	// Add flag for IMW name
	cmdLocation.Flags().StringVar(&imw, "imw", "", "IMW name")
	cmdLocation.MarkFlagRequired("imw")
	return cmdLocation
}

// The IMW name is provided by the --imw flag (stored in the "imw" string
// variable), and the location name is a command line argument.
// Labels to check are provided as additional arguments in key=value pairs.
func ensureLocation(cmdLocation *cobra.Command, args []string, cliOpts *genericclioptions.ConfigFlags) error {
	// locationName := args[0]
	// labels := args[1:]
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Print all flags and their values if verbosity level is at least 1
	cmdLocation.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	return nil
}
