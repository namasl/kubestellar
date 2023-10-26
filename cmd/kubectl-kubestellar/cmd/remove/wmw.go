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

// sub-command for removing a workload management workspace

package remove

import (
    "fmt"
    "flag"
//    "errors"

    "github.com/spf13/cobra"
    "github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
//	"k8s.io/klog/v2"
    "k8s.io/client-go/kubernetes"
)

// Create the Cobra sub-command for 'kubectl kubestellar remove wmw'
func NewCmdRemoveWmw() *cobra.Command {

	// Get config flags with default values
	cliOpts := genericclioptions.NewConfigFlags(true)

    // Make wmw command
    wmwCmd := &cobra.Command{
        Use:   "wmw",
        Short:  "Remove a KubeStellar workload management workspace",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
//            fmt.Printf("REMOVE WMW %s\n", args[0])
            err := removeWmw(cmd, cliOpts, args)
            return err
        },
    }

	// Make a new flag set named rmwmw
	fs := pflag.NewFlagSet("rmwmw", pflag.ExitOnError)

    // MAY BE POSSIBLE TO DO WITHOUT THIS
    // **** WHAT IS klog DOING WITH FLAGS? ****
//	klog.InitFlags(flag.CommandLine)
    // Add to Go flag set to fs **** WHAT IS THIS FOR
	fs.AddGoFlagSet(flag.CommandLine)

	// Add cliOpts flags to fs (flow from syntax is confusing)
	cliOpts.AddFlags(fs)

    // Add flags to our command
    wmwCmd.PersistentFlags().AddFlagSet(fs)
    // IS THIS FUNCTIONALLY IDENTICAL TO THE ABOVE?
//    cliOpts.AddFlags(wmwCmd.PersistentFlags())

    return wmwCmd
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

    vinfo, _ := client.Discovery().ServerVersion()
    fmt.Println(vinfo)

    return nil
//    return errors.New("rm wmw err")
}