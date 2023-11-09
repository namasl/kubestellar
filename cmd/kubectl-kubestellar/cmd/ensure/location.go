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
// Labels in key=value pairs are provided as command line arguments, for which
// we will ensure that these exist as labels in the Location and SyncTarget.

package ensure

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	v1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	v2alpha1 "github.com/kubestellar/kubestellar/pkg/apis/edge/v2alpha1"
	clientopts "github.com/kubestellar/kubestellar/pkg/client-options"
	clientset "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned"
)

var imw string // IMW workspace path

// Create the Cobra sub-command for 'kubectl kubestellar ensure location'
func newCmdEnsureLocation(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make location command
	cmdLocation := &cobra.Command{
		Use:     "location --imw <IMW_NAME> <LOCATION_NAME> <\"KEY=VALUE\" ...>",
		Aliases: []string{"loc"},
		Short:   "Ensure existence and configuration of an inventory listing for a WEC",
		// We actually require at least 2 arguments (location name and a label),
		// but more descriptive error messages will be provided by leaving this
		// set to 1.
		Args:    cobra.MinimumNArgs(1),
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
// - check that SyncTarget has an "id" label matching the Location name
// - ensure that SyncTarget has the labels provided by the user
// - check for Location of provided name in IMW, create if not
// - ensure that Location has the labels provided by the user
// - if Location "default" exists, delete it
func ensureLocation(cmdLocation *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string) error {
	locationName := args[0]
	labels := args[1:]
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Make sure user provided location name is valid
	err := checkLocationName(locationName, logger)
	if err != nil {
		return err
	}

	// Make sure user provided labels are valid
	err = checkLabelArgs(labels, logger)
	if err != nil {
		return err
	}

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

	// Check that SyncTarget exists and is configured, create/update if not
	err = verifyOrCreateSyncTarget(client, ctx, logger, locationName, labels)
	if err != nil {
		return err
	}

	// Check if Location exists and is configured, create/update if not
	err = verifyOrCreateLocation(client, ctx, logger, locationName, labels)
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

// Make sure user provided location name is valid
func checkLocationName(locationName string, logger klog.Logger) error {
	// ensure characters are valid
	matched, _ := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`, locationName)
	if !matched {
		err := errors.New("Location name must match regex '^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$'")
		logger.Error(err, fmt.Sprintf("Invalid location name %s", locationName))
		return err
	}
	// check for reserved words
	if locationName == "default" {
		err := errors.New("Location name 'default' may not be used")
		logger.Error(err, fmt.Sprintf("Invalid location name %s", locationName))
		return err
	}
	return nil
}

// Verify that user provided key=value arguments are valid
func checkLabelArgs(labels []string, logger klog.Logger) error {
	if len(labels) < 1 {
		err := errors.New("No labels provided")
		logger.Error(err, "At least one label must be provided")
		return err
	}
	// Iterate over labels
	for _, labelString := range labels {
		// Ensure the raw string contains a =
		if !strings.Contains(labelString, "=") {
			err := errors.New("Invalid label, missing '='")
			logger.Error(err, fmt.Sprintf("Invalid label, must format as \"key=value\": %s", labelString))
			return err
		}
		// Split substring on =
		labelSlice := strings.Split(labelString, "=")
		// We should have only a key and value now
		if len(labelSlice) != 2 {
			err := errors.New("Invalid label, must exactly one '='")
			logger.Error(err, fmt.Sprintf("Invalid label, must format as \"key=value\": %s", labelString))
			return err
		}
		key := labelSlice[0]
		value := labelSlice[1]
		// Make sure the key and value contain only valid characters
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9_./-]*$`, key)
		if !matched {
			err := errors.New("Key must match regex '^[a-zA-Z0-9][a-zA-Z0-9_./-]*$'")
			logger.Error(err, fmt.Sprintf("Invalid key %s in label \"%s\"", key, labelString))
			return err
		}
		matched, _ = regexp.MatchString(`^[a-zA-Z0-9]([a-zA-Z0-9_.-]{0,61}[a-zA-Z0-9])?$`, value)
		if !matched {
			err := errors.New("Value must match regex '^[a-zA-Z0-9]([a-zA-Z0-9_.-]{0,61}[a-zA-Z0-9])?$'")
			logger.Error(err, fmt.Sprintf("Invalid value %s in label \"%s\"", value, labelString))
			return err
		}
		// Make sure no invalid keys are passed
		if key == "id" {
			err := errors.New("Invalid key")
			logger.Error(err, "Key 'id' is handled internally and may not be specified")
			return err
		}
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
func verifyOrCreateSyncTarget(client *clientset.Clientset, ctx context.Context, logger klog.Logger, locationName string, labels []string) error {
	// Get the SyncTarget object
	syncTarget, err := client.EdgeV2alpha1().SyncTargets().Get(ctx, locationName, metav1.GetOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Found SyncTarget %s in workspace root:%s", locationName, imw))
		// Check that SyncTarget has an "id" label matching locationName
		err = verifySyncTargetId(syncTarget, client, ctx, logger, locationName)
		if err != nil {
			return err
		}
		// Check that SyncTarget has user provided key=value pairs, add them if not
		err = verifySyncTargetLabels(syncTarget, client, ctx, logger, locationName, labels)
		return err
	// TODO is converting err to a string the right way to check this?
	} else if err.Error() != fmt.Sprintf("synctargets.edge.kubestellar.io \"%s\" not found", locationName) {
		// Some error other than a non-existant SyncTarget
		logger.Info(fmt.Sprintf("Problem checking for SyncTarget %s in workspace root:%s", locationName, imw))
		return err
	}
	// SyncTarget does not exist, must create
	logger.Info(fmt.Sprintf("No SyncTarget %s in workspace root:%s, creating it", locationName, imw))

	syncTarget = &v2alpha1.SyncTarget {
		TypeMeta: metav1.TypeMeta {
			Kind: "SyncTarget",
			APIVersion: "edge.kubestellar.io/v2alpha1",
		},
		ObjectMeta: metav1.ObjectMeta {
			Name: locationName,
			Labels: map[string]string{"id": locationName},
		},
	}
	// Add any provided labels
	for _, labelString := range labels {
		// Split raw label string into key and value
		labelSlice := strings.Split(labelString, "=")
		key := labelSlice[0]
		value := labelSlice[1]
		syncTarget.ObjectMeta.Labels[key] = value
	}
	_, err = client.EdgeV2alpha1().SyncTargets().Create(ctx, syncTarget, metav1.CreateOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to create SyncTarget %s in workspace root:%s", locationName, imw))
		return err
	}

	return nil
}

// Make sure the SyncTarget has an id label matching locationName (update if not)
func verifySyncTargetId(syncTarget *v2alpha1.SyncTarget, client *clientset.Clientset, ctx context.Context, logger klog.Logger, locationName string) error {
	if syncTarget.ObjectMeta.Labels != nil {
		// We're missing a labels field, create it
		id := syncTarget.ObjectMeta.Labels["id"]
		if id == locationName {
			// id matches locationName, all good
			logger.Info(fmt.Sprintf("SyncTarget 'id' label matches %s", locationName))
			return nil
		}
		// ID label does not match locationName, update it
		logger.Info(fmt.Sprintf("SyncTarget %s 'id' label is '%s', changing to '%s'", locationName, id, locationName))
		syncTarget.ObjectMeta.Labels["id"] = locationName
	} else {
		// There are no labels, create it with id: locationName
		logger.Info(fmt.Sprintf("SyncTarget %s is missing labels, adding 'id'", locationName))
		syncTarget.ObjectMeta.Labels = map[string]string{"id": locationName}
	}

	// Apply updates to SyncTarget
	_, err := client.EdgeV2alpha1().SyncTargets().Update(ctx, syncTarget, metav1.UpdateOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to update SyncTarget %s in workspace root:%s", locationName, imw))
		return err
	}

	return nil
}

// Check that SyncTarget has user provided key=value pairs, add them if not
func verifySyncTargetLabels(syncTarget *v2alpha1.SyncTarget, client *clientset.Clientset, ctx context.Context, logger klog.Logger, locationName string, labels []string) error {
	updateSyncTarget := false // bool to see if we need to update SyncTarget
	// Check for labels missing or not matching those provide by user
	for _, labelString := range labels {
		// Split raw label string into key and value
		labelSlice := strings.Split(labelString, "=")
		key := labelSlice[0]
		value := labelSlice[1]
		// Make sure we have a labels field
		if syncTarget.ObjectMeta.Labels == nil {
			// There are no labels, create the label map with first label
			logger.Info("SyncTarget is missing labels, adding it")
			logger.Info(fmt.Sprintf("SyncTarget label %s=, updating value to %s", key, value))
			syncTarget.ObjectMeta.Labels = map[string]string{key: value}
			updateSyncTarget = true
			continue
		}
		valueCurrent := syncTarget.ObjectMeta.Labels[key]
		// Make sure label matches user provided value
		if valueCurrent != value {
			logger.Info(fmt.Sprintf("SyncTarget label %s=%s, updating value to %s", key, valueCurrent, value))
			syncTarget.ObjectMeta.Labels[key] = value
			updateSyncTarget = true
		} else {
			logger.Info(fmt.Sprintf("SyncTarget has label %s=%s", key, value))
		}
	}
	// Update SyncTarget if needed
	if updateSyncTarget {
		// Apply updates to SyncTarget
		_, err := client.EdgeV2alpha1().SyncTargets().Update(ctx, syncTarget, metav1.UpdateOptions{})
		if err != nil {
			logger.Info(fmt.Sprintf("Failed to update SyncTarget %s in workspace root:%s", locationName, imw))
			return err
		}
	}
	return nil
}

// Check if Location exists; if not, create one
func verifyOrCreateLocation(client *clientset.Clientset, ctx context.Context, logger klog.Logger, locationName string, labels []string) error {
	// Get the Location object
	location, err := client.EdgeV2alpha1().Locations().Get(ctx, locationName, metav1.GetOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Found Location %s in workspace root:%s", locationName, imw))
		// Check that Location has user provided key=value pairs, add them if not
		err = verifyLocationLabels(location, client, ctx, logger, locationName, labels)
		return err
	// TODO is converting err to a string the right way to check this?
	} else if err.Error() != fmt.Sprintf("locations.edge.kubestellar.io \"%s\" not found", locationName) {
		// Some error other than a non-existant SyncTarget
		logger.Info(fmt.Sprintf("Problem checking for Location %s in workspace root:%s", locationName, imw))
		return err
	}
	// Location does not exist, must create
	logger.Info(fmt.Sprintf("No Location %s in workspace root:%s, creating it", locationName, imw))

	location = &v2alpha1.Location {
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
	// Add any provided labels
	for _, labelString := range labels {
		// Split raw label string into key and value
		labelSlice := strings.Split(labelString, "=")
		key := labelSlice[0]
		value := labelSlice[1]
		if location.ObjectMeta.Labels != nil {
			// Add key=value
			location.ObjectMeta.Labels[key] = value
		} else {
			// No labels field exists, add the labels map along with this key=value
			location.ObjectMeta.Labels = map[string]string{key: value}
		}
	}
	_, err = client.EdgeV2alpha1().Locations().Create(ctx, location, metav1.CreateOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to create Location %s in workspace root:%s", locationName, imw))
		return err
	}

	return nil
}

// Check that Location has user provided key=value pairs, add them if not
func verifyLocationLabels(location *v2alpha1.Location, client *clientset.Clientset, ctx context.Context, logger klog.Logger, locationName string, labels []string) error {


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