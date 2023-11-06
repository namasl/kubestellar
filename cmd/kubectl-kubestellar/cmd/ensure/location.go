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

package ensure

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

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


// ....
// The IMW name is provided by the --imw flag (stored in the "imw" string
// variable), and the location name is a command line argument.
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
	// Set default context to "root"; we need to append the IMW name to the root server
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

	// Get the SyncTarget object
	syncTarget, err := client.EdgeV2alpha1().SyncTargets().Get(ctx, locationName, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to get SyncTarget %s", locationName))
		return err
	}
    logger.Info(fmt.Sprintf("Found SyncTarget %s in workspace root:%s", locationName, imw))

	// Get the Location object
	location, err := client.EdgeV2alpha1().Locations().Get(ctx, locationName, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to get Location %s", locationName))
		return err
	}
    logger.Info(fmt.Sprintf("Found Location %s in workspace root:%s", locationName, imw))

    fmt.Println("ST ST ST ST ST ST ST")
    fmt.Println(syncTarget)
    fmt.Println("LOC LOC LOC LOC LOC LOC")
    fmt.Println(location)


	return nil
}

// go to IMW

// check if API binding exists with
// $ kubectl get apibinding "edge.kubestellar.io"
// GET https://debian:1119/clusters/root:imw1/apis/apis.kcp.io/v1alpha1/apibindings/edge.kubestellar.io
// if this does not exist, then do (https://docs.kcp.io/kcp/main/reference/crd/apibindings.apis.kcp.io/)
// $ kubectl kcp bind apiexport root:espw:edge.kubestellar.io


// check for SyncTarget... we already have this above
// $ kubectl get synctargets.edge.kubestellar.io "$objname"
// if that doesn't exist, then create one....
//
// (cat <<EOF
// apiVersion: edge.kubestellar.io/v2alpha1
// kind: SyncTarget
// metadata:
//   name: "$objname"
//   labels:
//     id: "$objname"
// EOF
// ) | kubectl "${kubectl_flags[@]}" create -f - || {
//     echo "$0: Creation of SyncTarget failed" >&2
//     exit 3
// }


// check for Location... we have this above
// $ kubectl get locations.edge.kubestellar.io "$objname"
// (cat <<EOF
// apiVersion: edge.kubestellar.io/v2alpha1
// kind: Location
// spec:
//   resource: {group: edge.kubestellar.io, version: v2alpha1, resource: synctargets}
//   instanceSelector:
//     matchLabels: {"id":"$objname"}
// metadata:
//   name: "$objname"
// EOF
// ) | kubectl "${kubectl_flags[@]}" create -f - || {
//     echo "$0: Creation of SyncTarget failed" >&2
//     exit 3
// }
// fi


// see if Location named "default" exists, and delete if so
// if kubectl get locations.edge.kubestellar.io default
//     kubectl delete locations.edge.kubestellar.io default


// bash variable stlabs=
// $ kubectl get synctargets.edge.kubestellar.io ks-edge-cluster1 -o json | jq .metadata.labels
// gives the result:
// {
//   "env": "ks-edge-cluster1",
//   "id": "ks-edge-cluster1",
//   "location-group": "edge"
// }

// bash variable loclabs=
// $ kubectl get locations.edge.kubestellar.io ks-edge-cluster1 -o json | jq .metadata.labels
// gives the result:
// {
//   "env": "ks-edge-cluster1",
//   "location-group": "edge"
// }

// for SyncTarget/Location outputs above, make sure labelname=labelvalue pairs
// given at the command line match what is in the output. If not, overwrite them with
// kubectl label --overwrite synctargets.edge.kubestellar.io "$objname" "${key}=${val}"
// or
// kubectl label --overwrite locations.edge.kubestellar.io "$objname" "${key}=${val}"


// Not done in Bash script, but can also make sure the SyncTarget has the
// label "id" = objname