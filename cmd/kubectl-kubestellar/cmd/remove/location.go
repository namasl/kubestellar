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
    "flag"

    "github.com/spf13/cobra"
    "github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var cmdLocation = &cobra.Command{
    Use:   "location",
    Aliases: []string{"loc"},
    Short:  "Remove a KubeStellar location object",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("REMOVE LOCATION %s\n", args[0])
    },
}

func init() {
	// Make a new flag set named rmloc
	fs := pflag.NewFlagSet("rmloc", pflag.ExitOnError)

    // **** WHAT IS klog DOING WITH FLAGS? ****
//	klog.InitFlags(flag.CommandLine)
    // Add to Go flag set to fs **** WHAT IS THIS FOR
	fs.AddGoFlagSet(flag.CommandLine)

	// Get config flags with default values
	cliOpts := genericclioptions.NewConfigFlags(true)
	// Add cliOpts flags to fs (flow from syntax is confusing)
	cliOpts.AddFlags(fs)

    // Add flags to our command
    cmdLocation.PersistentFlags().AddFlagSet(fs)
}

// Perform the actual location removal
func removeLocation() error {
    return nil
}