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
// --with-kube is a required flag which determines if root:compute APIBindings are needed.

package ensure

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	plugin "github.com/kubestellar/kubestellar/pkg/cliplugins/kubestellar/ensure"
)

var withKube bool // Variable for --with-kube flag

// Create the Cobra sub-command for 'kubectl kubestellar ensure wds'
func newCmdEnsureWDS(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make wds command
	cmdWDS := &cobra.Command{
		Use:     "wds <WDS_NAME> [--with-kube=<TRUE/FALSE>]",
		Aliases: []string{"wmw"},
		Short:   "Ensure existence and configuration of a workload description space (WDS, formerly WMW)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := ensureWDS(cmd, args, cliOpts)
			return err
		},
	}

	// Add flag for
	cmdWDS.Flags().BoolVar(&withKube, "with-kube", true, "Include Kubernetes API bindings")
	return cmdWDS
}

// Perform validation of workload description space (WDS). The user will provide
// the WDS name as a command-line argument, along with a boolean for --with-kube.
// This function will:
//   - At the root KCP level, check for a WDS workspace having the user provided
//     name, create if needed
//   - At the root KCP level, check for an APIBinding "bind-espw" with export path
//     "root:espw" and export name "edge.kubestellar.io"
//   - If --with-kube is true, ensure a list of APIBindings exist with export path
//     "root:compute" (create any that are missing). If --with-kube is false, make
//     sure none of these exist (delete as needed).
func ensureWDS(cmdWDS *cobra.Command, args []string, cliOpts *genericclioptions.ConfigFlags) error {
	wdsName := args[0] // name of WDS
	ctx := cmdWDS.Context()
	logger := klog.FromContext(ctx)

	// Print all flags and their values if verbosity level is at least 1
	cmdWDS.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	// Make sure user provided WDS name is valid
	err := plugin.CheckWdsName(wdsName)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Invalid WDS name %s", wdsName))
		return err
	}

	// Set context to root if no context is provided with --context flag
	// Later on we will append the WDS name to the server provided by context
	if *cliOpts.Context == "" {
		configContext := "root"
		cliOpts.Context = &configContext
	}

	// Get client config from flags
	rootConfig, err := cliOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get config from flags")
		return err
	}

	// Create client-go instance from config
	kcpClient, err := kcpclientset.NewForConfig(rootConfig)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Check for WDS workspace, create if it does not exist
	// This function prints its own log messages, so no need to add any here.
	err = plugin.VerifyOrCreateWDS(kcpClient, ctx, wdsName)
	if err != nil {
		return err
	}

	// Make a config with the server set to work within WDS workspace
	wdsConfig := rest.CopyConfig(rootConfig)
	wdsConfig.Host += ":" + wdsName
	logger.V(1).Info(fmt.Sprintf("Set host to %s", wdsConfig.Host))

	// Create client to work in WDS workspace
	kcpClient, err = kcpclientset.NewForConfig(wdsConfig)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Check for APIBinding bind-espw, create if it does not exist
	// This function prints its own log messages, so no need to add any here.
	err = plugin.VerifyOrCreateAPIBinding(kcpClient, ctx, "bind-espw", "edge.kubestellar.io", "root:espw")
	if err != nil {
		return err
	}

	// Check for Kube APIBindings, add/remove as needed depending on withKube
	// This function prints its own log messages, so no need to add any here.
	err = plugin.EnsureKubeAPIBindings(kcpClient, ctx, withKube)
	if err != nil {
		return err
	}

	return nil
}
