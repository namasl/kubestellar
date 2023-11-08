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
// The location name is provided as a required command line argument.
// Optional key=value pairs are provided as command line arguments, for which
// we will ensure that these exist as labels in the Location and SyncTarget.

package ensure

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	//"k8s.io/client-go/tools/reference"
	"k8s.io/klog/v2"

	v1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	v2alpha1 "github.com/kubestellar/kubestellar/pkg/apis/edge/v2alpha1"
	clientopts "github.com/kubestellar/kubestellar/pkg/client-options"
	clientset "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned"
)

var imw string // IMW workspace path

// Create the Cobra sub-command for 'kubectl kubestellar remove location'
func newCmdEnsureLocation(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make location command
	cmdLocation := &cobra.Command{
		Use:     "location --imw <IMW_NAME> <LOCATION_NAME>",
		Aliases: []string{"loc"},
		Short:   "Ensure existence and configuration of an inventory listing for a WEC",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := ensureLocation(cmd, cliOpts, args)
			return err
		},
	}

	// add flag for IMW workspace
	cmdLocation.Flags().StringVar(&imw, "imw", "", "IMW workspace")
	cmdLocation.MarkFlagRequired("imw")
	return cmdLocation
}

// The IMW name is provided by the --imw flag (stored in the "imw" string
// variable), and the location name is a command line argument.
// Labels to check are provided as additional arguments in key=value pairs.
// In this function we will:
// - work in the provided IMW workspace
// - check if APIBinding "edge.kubestellar.io" exists in IMW, create if not
// - check for SyncTarget of provided name in IMW, create if not
// - check for Location of provided name in IMW, create if not
// - if Location "default" exists, delete it
// - check that provided key=value pairs exist as labels in SyncTarget and Location
// - check that SyncTarget has an "id" label matching the Location name
func ensureLocation(cmdLocation *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string) error {
	locationName := args[0]
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Print all flags and their values if verbosity level is at least 1
	cmdLocation.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	// Options for IMW workspace
	imwClientOpts := clientopts.NewClientOpts("imw", "access to the IMW workspace")
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
    kcpClient, err := kcpclientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Check that APIBinding exists, create if not
	err = verifyOrCreateAPIBinding(kcpClient, ctx, logger)
	if err != nil {
		return err
	}

	// Create client-go instance from config
	client, err := clientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Check that SyncTarget exists, create if not
	err = verifyOrCreateSyncTarget(client, ctx, logger, locationName)
	if err != nil {
		return err
	}


	// Check if Location exists; if not, create one
	err = verifyOrCreateLocation(client, ctx, logger, locationName)
	if err != nil {
		return err
	}

	// Check if "default" Location exists, and delete it if so
	err = verifyNoDefaultLocation(client, ctx, logger)
	if err != nil {
		return err
	}

	return nil
}

// Check if APIBinding exists; if not, create one.
func verifyOrCreateAPIBinding(client *kcpclientset.Clientset, ctx context.Context, logger klog.Logger) error {
	// Get the APIBinding
	_, err := client.ApisV1alpha1().APIBindings().Get(ctx, "edge.kubestellar.io", metav1.GetOptions{})
	if err == nil {
    	logger.Info(fmt.Sprintf("Found APIBinding edge.kubestellar.io in workspace root:%s", imw))
		return nil
	}

	// APIBinding does not exist, must create
	logger.Info(fmt.Sprintf("No APIBinding edge.kubestellar.io in workspace root:%s", imw))

	apiBinding := v1alpha1.APIBinding {
		TypeMeta: metav1.TypeMeta {
			Kind: "apis.kcp.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta {
			Name: "edge.kubestellar.io",
		},
		Spec: v1alpha1.APIBindingSpec {
			Reference: v1alpha1.BindingReference {
					Export: &v1alpha1.ExportBindingReference {
						Name: "edge.kubestellar.io",
						Path: "root:espw",
				},
			},
		},
	}
	_, err = client.ApisV1alpha1().APIBindings().Create(ctx, &apiBinding, metav1.CreateOptions{})
	if err != nil {
    	logger.Error(err, fmt.Sprintf("Failed to create APIBinding in workspace root:%s", imw))
		return err
	}

	return nil
}

// Check if SyncTarget exists; if not, create one
// apiVersion: edge.kubestellar.io/v2alpha1
// kind: SyncTarget
// metadata:
//   name: "$objname"
//   labels:
//     id: "$objname"
func verifyOrCreateSyncTarget(client *clientset.Clientset, ctx context.Context, logger klog.Logger, locationName string) error {
	// Get the SyncTarget object
	_, err := client.EdgeV2alpha1().SyncTargets().Get(ctx, locationName, metav1.GetOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Found SyncTarget %s in workspace root:%s", locationName, imw))
		// Check that SyncTarget has an "id" label matching locationName
		// Check that SyncTarget has user provided key=value pairs
		return nil
	}
	// SyncTarget does not exist, must create
	logger.Info(fmt.Sprintf("No SyncTarget %s in workspace root:%s, creating it", locationName, imw))

	syncTarget := v2alpha1.SyncTarget {
		TypeMeta: metav1.TypeMeta {
			Kind: "SyncTarget",
			APIVersion: "edge.kubestellar.io/v2alpha1",
		},
		ObjectMeta: metav1.ObjectMeta {
			Name: locationName,
			Labels: map[string]string{"id": locationName},
		},
	}
	_, err = client.EdgeV2alpha1().SyncTargets().Create(ctx, &syncTarget, metav1.CreateOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to create SyncTarget %s in workspace root:%s", locationName, imw))
		return err
	}

	return nil
}

// apiVersion: edge.kubestellar.io/v2alpha1
// kind: Location
// spec:
//   resource: {group: edge.kubestellar.io, version: v2alpha1, resource: synctargets}
//   instanceSelector:
//     matchLabels: {"id":"$objname"}
// metadata:
//   name: "$objname"
// Check if Location exists; if not, create one
func verifyOrCreateLocation(client *clientset.Clientset, ctx context.Context, logger klog.Logger, locationName string) error {
	// Get the Location object
	_, err := client.EdgeV2alpha1().Locations().Get(ctx, locationName, metav1.GetOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Found Location %s in workspace root:%s", locationName, imw))
		// Check that Location has user provided key=value pairs
		return nil
	}
	// Location does not exist, must create
	logger.Info(fmt.Sprintf("No Location %s in workspace root:%s, creating it", locationName, imw))

	location := v2alpha1.Location {
		TypeMeta: metav1.TypeMeta {
			Kind: "Location",
			APIVersion: "edge.kubestellar.io/v2alpha1",
		},
		ObjectMeta: metav1.ObjectMeta {
			Name: locationName,
		},
		Spec: v2alpha1.LocationSpec {
			Resource: v2alpha1.GroupVersionResource {
				Group: "edge.kubestellar.io",
				Version: "v2alpha1",
				Resource: "synctargets",
			},
			InstanceSelector: &metav1.LabelSelector {
				MatchLabels: map[string]string{"id": locationName},
			},
		},
	}
	_, err = client.EdgeV2alpha1().Locations().Create(ctx, &location, metav1.CreateOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to create SyncTarget %s in workspace root:%s", locationName, imw))
		return err
	}

	return nil
}

// Check if default Location exists, delete it if so
func verifyNoDefaultLocation(client *clientset.Clientset, ctx context.Context, logger klog.Logger) error {
	// Check for "default" Location object
	_, err := client.EdgeV2alpha1().Locations().Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		// Check if error is due to the lack of a "default" location object (what we want)
		// TODO is converting err to a string the right way to check this?
		if err.Error() == "locations.edge.kubestellar.io \"default\" not found" {
			logger.Info(fmt.Sprintf("Verified no default Location in workspace root:%s", imw))
			return nil
		}
		// There is some error other than trying to get a non-existent object
		logger.Error(err, fmt.Sprintf("Could not check if default Location in workspace root:%s", imw))
		return err
	}

	// "default" Location exists, delete it
	logger.Info(fmt.Sprintf("Found default Location in workspace root:%s, deleting it", imw))
	err = client.EdgeV2alpha1().Locations().Delete(ctx, "default", metav1.DeleteOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to delete default Location in workspace root:%s", imw))
		return err
	}
	return nil
}



// bash variable stlabs=
// $ kubectl get synctargets.edge.kubestellar.io ks-edge-cluster1 -o json | jq .metadata.labels
// gives the result:
// {
//   "env": "ks-edge-cluster1",
//   "id": "ks-edge-cluster1",
//   "location-group": "edge"
// }
//
// bash variable loclabs=
// $ kubectl get locations.edge.kubestellar.io ks-edge-cluster1 -o json | jq .metadata.labels
// gives the result:
// {
//   "env": "ks-edge-cluster1",
//   "location-group": "edge"
// }
//
//
// for SyncTarget/Location outputs above, make sure labelname=labelvalue pairs
// given at the command line match what is in the output. If not, overwrite them with
// kubectl label --overwrite synctargets.edge.kubestellar.io "$objname" "${key}=${val}"
// or
// kubectl label --overwrite locations.edge.kubestellar.io "$objname" "${key}=${val}"
//
//
// Not done in Bash script, but can also make sure the SyncTarget has the
// label "id" = objname