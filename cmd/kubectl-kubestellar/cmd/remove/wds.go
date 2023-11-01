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

// Sub-command for removing a workload description space (WDS),
// formerly known as a workload management workspace (WMW).

package remove

import (
    "os"
    "fmt"
    "context"

    "github.com/spf13/cobra"
    "github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
//    "k8s.io/client-go/kubernetes"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/klog/v2"

    kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"
//    tenancyv1alpha1 "github.com/kcp-dev/kcp/pkg/client/clientset/versioned/typed/tenancy/v1alpha1"
    "github.com/kcp-dev/logicalcluster/v3"

	clientopts "github.com/kubestellar/kubestellar/pkg/client-options"
)

// Create the Cobra sub-command for 'kubectl kubestellar remove wds'
func newCmdRemoveWds(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
    // Make wds command
    cmdWds := &cobra.Command{
        Use:   "wds <WDS_NAME>",
        Aliases: []string{"wmw"},
        Short:  "Delete a workload description space (WDS, formerly WMW)",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            // At this point set silence usage to true, so that any errors
            // following do not result in the help being printed. We only
            // want the help to be displayed when the error is due to an
            // invalid command.
            cmd.SilenceUsage = true
            err := removeWds(cmd, cliOpts, args)
            return err
        },
    }

    return cmdWds
}

// Perform the actual workload management workspace removal
func removeWds(cmdWds *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string) error {
    wdsName := args[0]
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
	// Make a new flag set named rmwds
	fs := pflag.NewFlagSet("rmwds", pflag.ExitOnError)
	// Add cliOpts flags to fs (flow from syntax is confusing, goes -->)
	cliOpts.AddFlags(fs)
    // add fs to rootClientOpts
    rootClientOpts.AddFlags(fs)

    //fs.Parse(os.Args[1:])

    fmt.Println("****** NEWFLAGS ******")
	fs.VisitAll(func(flg *pflag.Flag) {
		logger.Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

    // Get client config from flags
//    config, err := cliOpts.ToRESTConfig()
    config, err := rootClientOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get config from flags")
		return err
	}

    // Create client-go instance from config
    //client, err := kubernetes.NewForConfig(config)
    client, err := kcpclientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

    // Go to root KCP workspace
    // kubectl ws "${kubectl_flags[@]}" root
 
    // Check that provided WDS exists
    // if kubectl "${kubectl_flags[@]}" get workspaces.tenancy.kcp.io "$wds_name" &>/dev/null

    // Delete WDS
    // kubectl "${kubectl_flags[@]}" delete workspaces.tenancy.kcp.io "$wds_name"

    fmt.Println("****** PATH ******")
    wdsPath := logicalcluster.Name("root").Path().Join(wdsName).String()
    fmt.Println(wdsPath)


    // fmt.Println("****** LIST ******")
    // list, err := client.TenancyV1alpha1().WorkspaceTypes().List(ctx, metav1.ListOptions{})
	// if err != nil {
	// 	logger.Error(err, "Failed to list workspace")
	// 	return err
	// }
    // fmt.Println(list)



    //get, err := client.TenancyV1alpha1().WorkspaceTypes().Get(ctx, wdsName, metav1.GetOptions{})
    get, err := client.TenancyV1alpha1().Workspaces().Get(ctx, wdsName, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Failed to get workspace")
		return err
	}
    fmt.Println("****** GET ******")
    fmt.Println(get)


    return nil
//    return errors.New("rm wds err")
}