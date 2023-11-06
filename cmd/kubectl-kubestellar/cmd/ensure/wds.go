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
    "fmt"
    "context"

    "github.com/spf13/cobra"
    "github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/klog/v2"

    kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	clientopts "github.com/kubestellar/kubestellar/pkg/client-options"
)
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

    // Get WDS KCP workspace
    workspace, err := client.TenancyV1alpha1().Workspaces().Get(ctx, wdsName, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to get workspace %s", wdsName))
		return err
	}
    logger.Info(fmt.Sprintf("Found workspace %s", wdsName))

    fmt.Println("WS WS WS WS WS WS")
    fmt.Println(workspace)

    return nil
}