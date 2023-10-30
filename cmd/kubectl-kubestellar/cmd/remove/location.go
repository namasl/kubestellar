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

// sub-command for removing a location

package remove

import (
    "fmt"

    "github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var cmdLocation = &cobra.Command{
    Use:   "location",
    Aliases: []string{"loc"},
    Short:  "Delete an inventory entry for a given WEC",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("REMOVE LOCATION %s\n", args[0])
    },
}

// Create the Cobra sub-command for 'kubectl kubestellar remove location'
func newCmdRemoveLocation(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
    // Make wmw command
    cmdLocation := &cobra.Command{
        Use:   "location <LOCATION_NAME>",
        Aliases: []string{"loc"},
        Short:  "Delete an inventory entry for a given WEC",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            err := removeLocation(cmd, cliOpts, args)
            return err
        },
    }

    return cmdLocation
}

// Perform the actual location removal
func removeLocation(wmwCmd *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string) error {
    fmt.Printf("REMOVE LOC %s\n", args[0])
    return nil
}