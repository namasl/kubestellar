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

    "github.com/spf13/cobra"
    "github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

var wmwCmd = &cobra.Command{
    Use:   "wmw",
    Short:  "Remove a KubeStellar workload management workspace",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        fmt.Println("WMW")
        return nil
    },
}



func init() {
	// Make a newflag set named rmWmw
	fs := pflag.NewFlagSet("rmWmw", pflag.ExitOnError)

	klog.InitFlags(flag.CommandLine)
	fs.AddGoFlagSet(flag.CommandLine)

	// get config flags with default values
	cliOpts := genericclioptions.NewConfigFlags(true)
	// add cliOpts flags to fs (flow from syntax is confusing)
	cliOpts.AddFlags(fs)
}

func removeWmw() {
}