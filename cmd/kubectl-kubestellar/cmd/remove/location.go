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

// Sub-command for removing a location.
// The IMW is provided by the required --imw flag, and the location name is
// given as a command line argument. The SyncTarget and Location object within
// the IMW will be deleted.

package remove

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

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
		RunE:    func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := removeLocation(cmd, cliOpts, args)
			return err
		},
	}

	// Add flag for IMW workspace
	cmdLocation.Flags().StringVar(&imw, "imw", "", "IMW workspace")
	cmdLocation.MarkFlagRequired("imw")
	return cmdLocation
}

// Delete SyncTarget and Location from IMW.
// The IMW name is provided by the --imw flag (stored in the "imw" string
// variable), and the location name is a command line argument.
func removeLocation(cmdLocation *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string) error {
	locationName := args[0]
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Print all flags and their values if verbosity level is at least 1
	cmdLocation.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	// Options for IMW workspace
	imwClientOpts := clientopts.NewClientOpts("imw", "Access to the IMW workspace")
	// Set default context to "root"; we will need to append the IMW name to the root server
	imwClientOpts.SetDefaultCurrentContext("root")

	// Get client config from flags
	config, err := imwClientOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get config from flags")
		return err
	}

	// Update host to work on objects within IMW workspace
	config.Host += ":" + imw
	logger.V(1).Info(fmt.Sprintf("Set host to %s", config.Host))

	// Create client-go instance from config
	client, err := clientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Delete the SyncTarget object
	err = deleteSyncTarget(client, ctx, logger, locationName)
	if err != nil {
		return err
	}

	// Delete the Location object
	err = deleteLocation(client, ctx, logger, locationName)
	if err != nil {
		return err
	}

	return nil
}

// Delete a SyncTarget, don't return an error if it doesn't exist
func deleteSyncTarget(client *clientset.Clientset, ctx context.Context, logger klog.Logger, syncTargetName string) error {
	// Delete the workspace
	err := client.EdgeV2alpha1().SyncTargets().Delete(ctx, syncTargetName, metav1.DeleteOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Removed SyncTarget %s from workspace root:%s", syncTargetName, imw))
		return err
	} else if ! apierrors.IsNotFound(err) {
		// Some error other than a non-existant workspace
		logger.Info(fmt.Sprintf("Problem removing SyncTarget %s from workspace root:%s", syncTargetName, imw))
		return err
	}
	logger.Info(fmt.Sprintf("Verified no SyncTarget %s in workspace root:%s", syncTargetName, imw))
	return nil
}

// Delete a Location, don't return an error if it doesn't exist
func deleteLocation(client *clientset.Clientset, ctx context.Context, logger klog.Logger, locationName string) error {
	// Delete the workspace
	err := client.EdgeV2alpha1().Locations().Delete(ctx, locationName, metav1.DeleteOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Removed Location %s from workspace root:%s", locationName, imw))
		return err
	} else if ! apierrors.IsNotFound(err) {
		// Some error other than a non-existant workspace
		logger.Info(fmt.Sprintf("Problem removing Location %s from workspace root:%s", locationName, imw))
		return err
	}
	logger.Info(fmt.Sprintf("Verified no Location %s in workspace root:%s", locationName, imw))
	return nil
}