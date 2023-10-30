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

// Sub-command for removing a workload management workspace (WMW),
// also known as a workload description space (WDS)

package remove

import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
    "k8s.io/client-go/kubernetes"
)

// Create the Cobra sub-command for 'kubectl kubestellar remove wmw'
func newCmdRemoveWmw() *cobra.Command {

	// Get config flags with default values
	cliOpts := genericclioptions.NewConfigFlags(true)

    // Make wmw command
    cmdWmw := &cobra.Command{
        Use:   "wmw <WMW_NAME>",
        Aliases: []string{"wds"},
        Short:  "Delete a workload management workspace/description space (WMW/WDS)",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            err := removeWmw(cmd, cliOpts, args)
            return err
        },
    }

	// Make a new flag set named rm
	fs := pflag.NewFlagSet("rmwmw", pflag.ExitOnError)
	// Add cliOpts flags to fs (flow from syntax is confusing)
	cliOpts.AddFlags(fs)
    // Add flags to our command; make these persistent (available to this
    // command and all sub-commands)
    cmdWmw.PersistentFlags().AddFlagSet(fs)

    return cmdWmw
}

// Perform the actual workload management workspace removal
func removeWmw(wmwCmd *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string) error {

    fmt.Printf("REMOVE WMW %s\n", args[0])

    // Get client config from flags
    config, err := cliOpts.ToRESTConfig()
	if err != nil {
//		logger.Error(err, "Failed to get client from flags")
		return err
	}

    // Create client-go instance from config
    client, err := kubernetes.NewForConfig(config)
	if err != nil {
//		logger.Error(err, "Failed create client-go instance")
		return err
	}

    //resource, err := client.CoreV1().

    // go to root KCP workspace
    // kubectl ws "${kubectl_flags[@]}" root
 
    // check that provided WMW exists
    // if kubectl "${kubectl_flags[@]}" get workspaces.tenancy.kcp.io "$wmw_name" &>/dev/null

    // delete WMW
    // kubectl "${kubectl_flags[@]}" delete workspaces.tenancy.kcp.io "$wmw_name"

    fmt.Println("done")

    return nil
//    return errors.New("rm wmw err")
}