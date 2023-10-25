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

// This is the root for kubectl-kubestellar.


package cmd

import (
	"fmt"
	"os"
	"errors"

	"github.com/spf13/cobra"

//	"k8s.io/cli-runtime/pkg/genericclioptions"
//	"k8s.io/klog/v2"

	"github.com/kubestellar/kubestellar/cmd/kubectl-kubestellar/cmd/ensure"
	"github.com/kubestellar/kubestellar/cmd/kubectl-kubestellar/cmd/remove"
)

// root kubestellar command
var rootCmd = &cobra.Command{
	Use:	"kubestellar",
	Short:	"KubeStellar plugin for kubectl",
	Long:	`KubeStellar is a flexible solution for challenges associated with multi-cluster 
configuration management for edge, multi-cloud, and hybrid cloud.
This command provides the kubestellar sub-command for kubectl.`,
	// TODO: without specifying this, when no args are present the help does
    // not get printed after the error.
    Args:  cobra.ExactArgs(1),
    // If an invalid sub-command is sent, the function in RunE will execute.
    // Use this to inform of invalid arguments, and return an error.
    RunE: func(cmd *cobra.Command, args []string) error {
        if len(args) > 0 {
            return errors.New(fmt.Sprintf("Invalid kubestellar sub-command: %s\n", args[0]))
        } else {
			// TODO: The help does not get printed after this message. If
			// specifying  "Args: cobra.ExactArgs(1)" then the help does get
			// printed, but this line will never run.
            return errors.New(fmt.Sprintf("Missing kubestellar sub-command\n"))
        }
    },
}

// add sub-commands to root
func init() {
    rootCmd.AddCommand(ensure.EnsureCmd)
    rootCmd.AddCommand(remove.RemoveCmd)

/*
	// Make a newflag set named root
	fs := pflag.NewFlagSet("root", pflag.ExitOnError)

	klog.InitFlags(flag.CommandLine)
	fs.AddGoFlagSet(flag.CommandLine)

	// get config flags with default values
	cliOpts := genericclioptions.NewConfigFlags(true)
	// add cliOpts flags to fs (syntax is confusing)
	cliOpts.AddFlags(fs)
*/
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}