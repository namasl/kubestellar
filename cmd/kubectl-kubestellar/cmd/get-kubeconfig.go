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

// This is the sub-command for getting the KubeStellar kubectl configuration.
// "get-external-kubeconfig" is used when running externally to the cluster hosting Kubestellar.
// "get-internal-kubeconfig" is used when running inside the same cluster as Kubestellar.
// --output (-o) is a required flag for providing an output filename for the config file.

package cmd

import (
    "context"
	"errors"
    "fmt"
	"flag"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
)

const ksContext = "ks-core" // Context for interacting with KubeStellar componnet pods
const ksNamespace = "kubestellar" // Namespace the KubeStellar pods are running in
const ksSelector = "app=kubestellar" // Selector (label query) for KubeStellar pods

var fname string // Filename/path for output configuration file (--output flag)

// Create the Cobra sub-command for 'kubectl kubestellar get-external-kubeconfig'
func newGetExternalKubeconfig(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make wds command
	cmdGetExternalKubeconfig := &cobra.Command{
		Use:     "get-external-kubeconfig",
		Aliases: []string{"gek"},
		Short:   "Get KubeStellar kubectl configuration when external to host cluster",
		Args:    cobra.ExactArgs(0),
		RunE:    func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := getKubeconfig(cmd, cliOpts, args, false)
			return err
		},
	}

	// Add required flag for output filename (--output or -o)
	cmdGetExternalKubeconfig.Flags().StringVarP(&fname, "output", "o", "", "Output path/filename")
	cmdGetExternalKubeconfig.MarkFlagRequired("output")
	return cmdGetExternalKubeconfig
}

// Create the Cobra sub-command for 'kubectl kubestellar get-internal-kubeconfig'
func newGetInternalKubeconfig(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
	// Make wds command
	cmdGetInternalKubeconfig := &cobra.Command{
		Use:     "get-internal-kubeconfig",
		Aliases: []string{"gik"},
		Short:   "Get KubeStellar kubectl configuration from inside same cluster",
		Args:    cobra.ExactArgs(0),
		RunE:    func(cmd *cobra.Command, args []string) error {
			// At this point set silence usage to true, so that any errors
			// following do not result in the help being printed. We only
			// want the help to be displayed when the error is due to an
			// invalid command.
			cmd.SilenceUsage = true
			err := getKubeconfig(cmd, cliOpts, args, true)
			return err
		},
	}

	// Add required flag for output filename (--output or -o)
	cmdGetInternalKubeconfig.Flags().StringVarP(&fname, "output", "o", "", "Output path/filename")
	cmdGetInternalKubeconfig.MarkFlagRequired("output")
	cmdGetInternalKubeconfig.MarkFlagFilename("output")
	return cmdGetInternalKubeconfig
}

func init() {
	// Get config flags with default values.
	// Passing "true" will "use persistent client config, rest mapper,
	// discovery client, and propagate them to the places that need them,
	// rather than instantiating them multiple times."
	cliOpts := genericclioptions.NewConfigFlags(true)
	// Make a new flag set named getKubeconfig
	fs := pflag.NewFlagSet("getKubeconfig", pflag.ExitOnError)
	// Add cliOpts flags to fs (flow from syntax is confusing, goes -->)
	cliOpts.AddFlags(fs)
	// Add logging flags to fs
	fs.AddGoFlagSet(flag.CommandLine)
	// Add flags to our command; make these persistent (available to this
	// command and all sub-commands)
	rootCmd.PersistentFlags().AddFlagSet(fs)

	// Add sub-commands
	rootCmd.AddCommand(newGetExternalKubeconfig(cliOpts))
	rootCmd.AddCommand(newGetInternalKubeconfig(cliOpts))
}

func getKubeconfig(cmdGetKubeconfig *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string, isInternal bool) error {
	ctx := context.Background()
	logger := klog.FromContext(ctx)

	// Print all flags and their values if verbosity level is at least 1
	cmdGetKubeconfig.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

	// Get client config from flags
	config, err := cliOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get config from flags")
		return err
	}

	// Create client-go instance from config
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

	// Get name of KubeStellar server pod
	serverPodName, err := getServerPodName(client, ctx, logger)
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("Found KubeStellar server pod %s", serverPodName))


    return nil
}

// Get name of pod running KubeStellar server
func getServerPodName(client *kubernetes.Clientset, ctx context.Context, logger klog.Logger) (string, error) {
	// Get list of pods matching selector in given namespace
	podList, err := client.CoreV1().Pods(ksNamespace).List(ctx, metav1.ListOptions{LabelSelector: ksSelector})
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return "", err
	}

	// Make sure we get one matching pod
	if len(podList.Items) == 0 {
		err = errors.New("No server pods")
		logger.Error(err, "Could not find a server pod in namespace %s with selector %s", ksNamespace, ksSelector)
		return "", err
	} else if len(podList.Items) > 1 {
		err = errors.New("More than one server pod")
		logger.Error(err, "Found %d server pods in namespace %s with selector %s", len(podList.Items), ksNamespace, ksSelector)
		return "", err
	}

	// Return pod name
	return podList.Items[0].Name, nil
}

func executeCommandInPod(client *kubernetes.Clientset, config *rest.Config, podName string, command []string,
	stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	req := client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(ksNamespace).SubResource("exec")

	option := &corev1.PodExecOptions{
		Command: command,
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}
	if stdin == nil {
		option.Stdin = false
	}

	req.VersionedParams(option, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	err = exec.Stream(remotecommand.StreamOptions{Stdin:  stdin, Stdout: stdout, Stderr: stderr})
	if err != nil {
		return err
	}

	return nil
}