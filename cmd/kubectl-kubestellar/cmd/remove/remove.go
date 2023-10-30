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

// This is the "remove" sub-command for kubestellar.

package remove

import (
    "fmt"
    "errors"

    "github.com/spf13/cobra"
    "github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var RemoveCmd = &cobra.Command{
    Use:    "remove",
    Aliases: []string{"rm"},
    Short:  "Remove a KubeStellar object",
//    Args:  cobra.ExactArgs(1),
    // If an invalid sub-command is sent, the function in RunE will execute.
    // Use this to inform of invalid arguments, and return an error.
    RunE: func(cmd *cobra.Command, args []string) error {
        if len(args) > 0 {
            return errors.New(fmt.Sprintf("Invalid sub-command for 'remove': %s\n", args[0]))
        } else {
            return errors.New(fmt.Sprintf("Missing sub-command for 'remove'\n"))
        }
    },
}

func init() {
	// Get config flags with default values
	cliOpts := genericclioptions.NewConfigFlags(true)
	// Make a new flag set named rm
	fs := pflag.NewFlagSet("rm", pflag.ExitOnError)
	// Add cliOpts flags to fs (flow from syntax is confusing)
	cliOpts.AddFlags(fs)
    // Add flags to our command; make these persistent (available to this
    // command and all sub-commands)
    RemoveCmd.PersistentFlags().AddFlagSet(fs)

    // add location sub-command
    RemoveCmd.AddCommand(newCmdRemoveLocation(cliOpts))
    // add wmw sub-command
    RemoveCmd.AddCommand(newCmdRemoveWmw(cliOpts))
}