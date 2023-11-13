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

// Sub-command for ensuring the existence of a workload description space (WDS),
// along with requisite APIBindings.
// The WDS name is given as a required command-line argument.
// --with-kube is a required flag which determines if Kube APIBindings are needed.

package ensure

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	tenancyv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/tenancy/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	clientopts "github.com/kubestellar/kubestellar/pkg/client-options"
)

var withKube bool

// Create the Cobra sub-command for 'kubectl kubestellar ensure wds'
func newCmdEnsureWds(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make wds command
	cmdWds := &cobra.Command{
		Use:   "wds <WDS_NAME>",
		Aliases: []string{"wmw"},
		Short:  "Ensure existence and configuration of a workload description space (WDS, formerly WMW)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := ensureWds(cmd, cliOpts, args)
			return err
		},
	}

	// Add flag for 
	cmdWds.Flags().BoolVar(&withKube, "with-kube", true, "Include API binding")
	cmdWds.MarkFlagRequired("with-kube")
	return cmdWds
}

// Perform validation of workload description space (WDS). The user will provide
// the WDS name as a command-line argument, along with a boolean for --with-kube.
// This function will:
// - At the root KCP level, check for a WDS workspace having the user provided
//   name, create if needed
// - At the root KCP level, check for an APIBinding "bind-espw" with export path
//   "root:espw" and export name "edge.kubestellar.io"
// - If --with-kube is true, ensure a list of APIBindings exist with export path
//   "root:compute" (create any that are missing). If --with-kube is false, make
//   sure none of these exist (delete as needed).
func ensureWds(cmdWds *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string) error {
	wdsName := args[0] // name of WDS
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Print all flags and their values if verbosity level is at least 1
	cmdWds.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	// Make sure user provided WDS name is valid
	err := checkLocationName(wdsName, logger)
	if err != nil {
		return err
	}

	// Options for WDS workspace
	wdsClientOpts := clientopts.NewClientOpts("wds", "Access to the WDS workspace")
	// Set default context to "root", later on we will append the WDS name to the root server
	wdsClientOpts.SetDefaultCurrentContext("root")

	// Get client config from flags
	config, err := wdsClientOpts.ToRESTConfig()
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

	// Check for WDS workspace, create if it does not exist
	err = verifyOrCreateWDS(client, ctx, logger, wdsName)
	if err != nil {
		return err
	}

	// Update host to work on objects within WDS workspace
	config.Host += ":" + wdsName
	logger.V(1).Info(fmt.Sprintf("Set host to %s", config.Host))

	// Update client to work in WDS workspace
	client, err = kcpclientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Check for APIBinding bind-espw, create if it does not exist
	err = verifyOrCreateAPIBinding(client, ctx, logger, "bind-espw", "edge.kubestellar.io", "root:espw")
	if err != nil {
		return err
	}

	// Check for Kube APIBindings, add/remove as needed depending on withKube
	err = verifyKubeAPIBindings(client, ctx, logger)
	if err != nil {
		return err
	}

	return nil
}

// Make sure user provided WDS name is valid
func checkWdsName(wdsName string, logger klog.Logger) error {
	// ensure characters are valid
	matched, _ := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`, wdsName)
	if !matched {
		err := errors.New("WDS name must match regex '^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$'")
		logger.Error(err, fmt.Sprintf("Invalid WDS name %s", wdsName))
		return err
	}
	return nil
}

// Check if WDS workspace exists, and if not create it
func verifyOrCreateWDS(client *kcpclientset.Clientset, ctx context.Context, logger klog.Logger, wdsName string) error {
	// Check if WDS workspace exists
	_, err := client.TenancyV1alpha1().Workspaces().Get(ctx, wdsName, metav1.GetOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Found WDS workspace %s", wdsName))
		return err
	}
	if ! apierrors.IsNotFound(err) {
		// Some error other than a non-existant workspace
		logger.Error(err, fmt.Sprintf("Error checking for WDS %s", wdsName))
		return err
	}

	// WDS workspace does not exist, create it
	logger.Info(fmt.Sprintf("No WDS workspace %s, creating it", wdsName))

	workspace := &tenancyv1alpha1.Workspace {
		TypeMeta: metav1.TypeMeta {
			Kind: "Workspace",
			APIVersion: "tenancy.kcp.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta {
			Name: wdsName,
		},
	}
	_, err = client.TenancyV1alpha1().Workspaces().Create(ctx, workspace, metav1.CreateOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to create WDS workspace %s", wdsName))
		return err
	}

	// Wait for workspace to become ready
	wait.Poll(time.Millisecond*100, time.Second*5, func() (bool, error) {
		// See if we can get new workspace
		if _, err := client.TenancyV1alpha1().Workspaces().Get(ctx, wdsName, metav1.GetOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				// Failed to get due to not found, try until timeout
				return false, nil
			}
			// Some error happened beyond not finding the workspace
			return false, err
		}
		// We got the workspace, we're good to go
		return true, nil
	})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem waiting for WDS workspace %s", wdsName))
		return err
	}

	return nil
}

// Check for Kube APIBindings
// If withKube is true, create any bindings that don't exist
// If withKube is false, delete any bindings that exist
func verifyKubeAPIBindings(client *kcpclientset.Clientset, ctx context.Context, logger klog.Logger) error {
	// APIBindings to check
	binds := []string {
		"kubernetes",
		"apiregistration.k8s.io",
		"apps",
		"autoscaling",
		"batch",
		"core.k8s.io",
		"cluster-core.k8s.io",
		"discovery.k8s.io",
		"flowcontrol.apiserver.k8s.io",
		"networking.k8s.io",
		"cluster-networking.k8s.io",
		"node.k8s.io",
		"policy",
		"scheduling.k8s.io",
		"storage.k8s.io",
		"cluster-storage.k8s.io",
	}
	// Iterate over bindings
	for _, exportName := range binds {
		bindName := "bind-" + exportName
		if withKube {
			// Make sure these bindings exist
			err := verifyOrCreateAPIBinding(client, ctx, logger, bindName, exportName, "root:compute")
			if err != nil {
				return err
			}
		} else {
			// Remove these bindings if they exist
			err := deleteAPIBinding(client, ctx, logger, bindName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Delete an API binding, don't return an error if it doesn't exist
func deleteAPIBinding(client *kcpclientset.Clientset, ctx context.Context, logger klog.Logger, bindName string) error {
	// Delete the APIBinding
	err := client.ApisV1alpha1().APIBindings().Delete(ctx, bindName, metav1.DeleteOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Removed APIBinding %s", bindName))
		return err
	} else if ! apierrors.IsNotFound(err) {
		// Some error other than a non-existant APIBinding
		logger.Info(fmt.Sprintf("Problem removing APIBinding %s", bindName))
		return err
	}
	logger.Info(fmt.Sprintf("Verified no APIBinding %s", bindName))
	return nil
}