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

// Sub-command for ensuring the existence and configuration of a WDS.

package ensure

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	tenancyv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/tenancy/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	//"github.com/kubestellar/kubestellar/pkg/apis/meta/v1alpha1"
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

// Perform validation of workload management workspace
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

	// Options for root workspace
	rootClientOpts := clientopts.NewClientOpts("root", "access to the root workspace")
	// set default context to "root"
	rootClientOpts.SetDefaultCurrentContext("root")

	// Get client config from flags
	config, err := rootClientOpts.ToRESTConfig()
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

	// Check for APIBinding bind-espw, create if it does not exist

	// Check for Kube APIBindings
	// If withKube is true, create any bindings that don't exist
	// If withKube is false, delete any bindings that exist

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

func verifyOrCreateWDS(client *kcpclientset.Clientset, ctx context.Context, logger klog.Logger, wdsName string) error {
	// Check if WDS workspace exists
	_, err := client.TenancyV1alpha1().Workspaces().Get(ctx, wdsName, metav1.GetOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Found WDS workspace %s", wdsName))
		return err
	}
	if err.Error() != fmt.Sprintf("workspaces.tenancy.kcp.io \"%s\" not found", wdsName) {
		// Some error other than a non-existant workspace
		logger.Error(err, fmt.Sprintf("Error checking for WDS %s", wdsName))
		return err
	}

	// WDS workspace does not exist, create it
	logger.Error(err, fmt.Sprintf("No WDS workspace %s, creating it", wdsName))

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
	return nil
}

// apiVersion: apis.kcp.io/v1alpha1
// kind: APIBinding
// metadata:
//   name: BINDING_NAME
// spec:
//   reference:
//	 export:
//	   path: PATH_NAME
//	   name: EXPORT_NAME
// Create an APIBinding
func createAPIBinding() error {
	return nil
}