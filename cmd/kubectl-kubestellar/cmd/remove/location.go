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

// sub-command for removing a location

package remove

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"
	"github.com/kcp-dev/logicalcluster/v3"

	clientopts "github.com/kubestellar/kubestellar/pkg/client-options"
	clientset "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned"
)

var imw string // IMW workspace path

// Create the Cobra sub-command for 'kubectl kubestellar remove location'
func newCmdRemoveLocation(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make location command
	cmdLocation := &cobra.Command{
		Use:     "location --imw <IMW_NAME> <LOCATION_NAME>",
		Aliases: []string{"loc"},
		Short:   "Delete an inventory entry for a given WEC",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := removeLocation(cmd, cliOpts, args)
			return err
		},
	}

	// add flag for IMW workspace

	cmdLocation.Flags().StringVar(&imw, "imw", "", "IMW workspace")
	cmdLocation.MarkFlagRequired("imw")
	return cmdLocation
}

// Perform the actual location removal
func removeLocation(cmdLocation *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string) error {
	locationName := args[0]
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Print all flags and their values if verbosity level is at least 1
	cmdLocation.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	imwPath := logicalcluster.Name("root").Path().Join(imw).String()

	// Options for IMW workspace
	imwClientOpts := clientopts.NewClientOpts("imw", "access to the IMW workspace")
	// set default context to "root"
	imwClientOpts.SetDefaultCurrentContext("root")
	// Make a new flag set named rmloc
	fs := pflag.NewFlagSet("rmloc", pflag.ExitOnError)
	// Add cliOpts flags to fs (flow from syntax is confusing, goes -->)
	cliOpts.AddFlags(fs)
	// add fs to imwClientOpts
	imwClientOpts.AddFlags(fs)



	// Options for location
	locClientOpts := clientopts.NewClientOpts("loc", "access to the IMW workspace")
	// set default context to "root"
	locClientOpts.SetDefaultCurrentContext("workspace.kcp.io/current")
	// add fs to locClientOpts
	locClientOpts.AddFlags(fs)

	// Get client config from flags
	config, err := imwClientOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get config from flags")
		return err
	}

	// Create client-go instance from config
	client, err := kcpclientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	fmt.Printf("REMOVE LOC %s, IMW=%s\n", locationName, imwPath)

	fmt.Println("****** GET ******")
	get, err := client.TenancyV1alpha1().Workspaces().Get(ctx, imw, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Failed to get workspace")
		return err
	}
	fmt.Println(get)

	// Get client config from flags
	configLoc, err := locClientOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get config from flags")
		return err
	}

	clientLoc, err := clientset.NewForConfig(configLoc)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	fmt.Println("******########## GET #############******")
	//getLoc, err := clientLoc.EdgeV2alpha1().EdgePlacements().Get(ctx, locationName, metav1.GetOptions{})
	getLoc, err := clientLoc.EdgeV2alpha1().SyncTargets().Get(ctx, locationName, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Failed to get")
		return err
	}
	fmt.Println(getLoc)

	//kubectl "${kubectl_flags[@]}" delete synctargets.edge.kubestellar.io "$objname"

	//kubectl "${kubectl_flags[@]}" delete locations.edge.kubestellar.io "$objname"

	return nil
}
