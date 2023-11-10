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

// This is the "ensure" sub-command for kubestellar.

package ensure

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"


	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"

	v1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"
)

// Create Cobra sub-command for 'kubectl kubestellar ensure'
var EnsureCmd = &cobra.Command{
	Use:	"ensure",
	Short:  "Ensure a KubeStellar object is correctly configured",
//	Args:  cobra.ExactArgs(1),
	// If an invalid sub-command is sent, the function in RunE will execute.
	// Use this to inform of invalid arguments, and return an error.
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New(fmt.Sprintf("Invalid sub-command for 'ensure': %s\n", args[0]))
		} else {
			return errors.New(fmt.Sprintf("Missing sub-command for 'ensure'\n"))
		}
	},
}

func init() {
	// Get config flags with default values.
	// Passing "true" will "use persistent client config, rest mapper,
	// discovery client, and propagate them to the places that need them,
	// rather than instantiating them multiple times."
	cliOpts := genericclioptions.NewConfigFlags(true)
	// Make a new flag set named en
	fs := pflag.NewFlagSet("en", pflag.ExitOnError)
	// Add cliOpts flags to fs (flow from syntax is confusing, goes -->)
	cliOpts.AddFlags(fs)

	// Add logging flags to fs
	fs.AddGoFlagSet(flag.CommandLine)
	// Add flags to our command; make these persistent (available to this
	// command and all sub-commands)
	EnsureCmd.PersistentFlags().AddFlagSet(fs)

	// Add location sub-command
	EnsureCmd.AddCommand(newCmdEnsureLocation(cliOpts))
	// Add wds sub-command
	EnsureCmd.AddCommand(newCmdEnsureWds(cliOpts))
}

// Check if an APIBinding exists, create if not
func verifyOrCreateAPIBinding(client *kcpclientset.Clientset, ctx context.Context, logger klog.Logger, bindName, exportName, exportPath string) error {
	// Get the APIBinding
	_, err := client.ApisV1alpha1().APIBindings().Get(ctx, bindName, metav1.GetOptions{})
	if err == nil {
		logger.Info(fmt.Sprintf("Found APIBinding %s", bindName))
		return err
	} else if ! apierrors.IsNotFound(err) {
		// Some error other than a non-existant APIBinding
		logger.Info(fmt.Sprintf("Problem checking for APIBinding %s", bindName))
		return err
	}

	// APIBinding does not exist, create it
	logger.Info(fmt.Sprintf("No APIBinding %s, creating it", bindName))

	apiBinding := v1alpha1.APIBinding {
		TypeMeta: metav1.TypeMeta {
			Kind: "APIBinding",
			APIVersion: "apis.kcp.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta {
			Name: bindName,
		},
		Spec: v1alpha1.APIBindingSpec {
			Reference: v1alpha1.BindingReference {
					Export: &v1alpha1.ExportBindingReference {
						Path: exportPath,
						Name: exportName,
				},
			},
		},
	}
	_, err = client.ApisV1alpha1().APIBindings().Create(ctx, &apiBinding, metav1.CreateOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to create APIBinding %s", bindName))
		return err
	}

	// Wait for new APIBinding, or timeout
	wait.Poll(time.Millisecond*100, time.Second*5, func() (bool, error) {
		// See if we can get new APIBinding
		if _, err := client.ApisV1alpha1().APIBindings().Get(ctx, bindName, metav1.GetOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				// Failed to get due to not found, try until timeout
				return false, nil
			}
			// Some error happened beyond not finding the APIBinding
			return false, err
		}
		// We got the APIBinding, we're good to go
		return true, nil
	})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Problem waiting for APIBinding %s", bindName))
		return err
	}

	return nil
}