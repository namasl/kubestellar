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

package cmd

import (
	"fmt"
	"os"
//	"flag"

	"github.com/spf13/cobra"

//	"k8s.io/cli-runtime/pkg/genericclioptions"
//	"k8s.io/klog/v2"

	"github.com/kubestellar/kubestellar/cmd/kubectl-kubestellar/cmd/ensure"
)

var rootCmd = &cobra.Command{
	Use:   "kubestellar",
	Short: "KubeStellar plugin for kubectl",
	Long: `KubeStellar is a flexible solution for challenges associated with multi-cluster 
configuration management for edge, multi-cloud, and hybrid cloud.
This command provides the kubestellar sub-command for kubectl.`,
	SilenceErrors: false, // print errors
	SilenceUsage: false, // print usage when there is an error
	// This root command is not executable, and requires a sub-command. Thus,
	// there is no "Run" or "RunE" value.
}

/*func init() {
	fs := flag.NewFlagSet("klog", flag.PanicOnError)
	klog.InitFlags(fs)
	rootCmd.PersistentFlags().AddGoFlagSet(fs)
	cliOpts := genericclioptions.NewConfigFlags(false)
	cliOpts.AddFlags(rootCmd.PersistentFlags())

//	getlogCmd := getlogcmd.New(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}, cliOpts)
//	root.AddCommand(getlogCmd)
}*/

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(2)
}

func init() {
    rootCmd.AddCommand(ensure.EnsureCmd)
}

/*
func NewKubestellarCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "kubestellar",
		Short: "kubectl plugin for KubeStellar",
		Long: `KubeStellar is a flexible solution for challenges associated with multi-cluster 
configuration management for edge, multi-cloud, and hybrid cloud.
This command provides the kubestellar sub-command for kubectl.`,
//		SilenceUsage:  false,
//		SilenceErrors: false,
	}

	// setup klog
	fs := flag.NewFlagSet("klog", flag.PanicOnError)
	klog.InitFlags(fs)
	root.PersistentFlags().AddGoFlagSet(fs)
	cliOpts := genericclioptions.NewConfigFlags(false)
	cliOpts.AddFlags(root.PersistentFlags())


//	getlogCmd := getlogcmd.New(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}, cliOpts)
//	root.AddCommand(getlogCmd)

	return root
}
*/