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
    "context"

    "github.com/spf13/cobra"
    "github.com/spf13/pflag"


	"k8s.io/cli-runtime/pkg/genericclioptions"
//    "k8s.io/client-go/kubernetes"
    v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/klog/v2"

    kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"
//    tenancyv1alpha1 "github.com/kcp-dev/kcp/pkg/client/clientset/versioned/typed/tenancy/v1alpha1"
)

// Create the Cobra sub-command for 'kubectl kubestellar remove wmw'
func newCmdRemoveWmw(cliOpts *genericclioptions.ConfigFlags) *cobra.Command {
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

    return cmdWmw
}

// Perform the actual workload management workspace removal
func removeWmw(cmdWmw *cobra.Command, cliOpts *genericclioptions.ConfigFlags, args []string) error {
    wmwName := args[0]
    ctx := context.Background()
	logger := klog.FromContext(ctx)

    var opts v1.GetOptions

	cmdWmw.Flags().VisitAll(func(flg *pflag.Flag) {
		logger.V(1).Info(fmt.Sprintf("Command line flag %s=%s", flg.Name, flg.Value))
	})

    logger.Info(fmt.Sprintf("REMOVE WMW %s", wmwName))

    // Get client config from flags
    config, err := cliOpts.ToRESTConfig()
	if err != nil {
		logger.Error(err, "Failed to get client from flags")
		return err
	}

    // Create client-go instance from config
    //client, err := kubernetes.NewForConfig(config)
    client, err := kcpclientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Failed create client-go instance")
		return err
	}

    // go to root KCP workspace
    // kubectl ws "${kubectl_flags[@]}" root
 
    // check that provided WMW exists
    // if kubectl "${kubectl_flags[@]}" get workspaces.tenancy.kcp.io "$wmw_name" &>/dev/null
//nick@debian:~$ KUBECONFIG=ks-core.kubeconfig kubectl api-resources
//NAME                              SHORTNAMES   APIVERSION                        NAMESPACED   KIND
//workspaces                        ws           tenancy.kcp.io/v1alpha1           false        Workspace

    //resource, err := client.CoreV1alpha1().RESTClient().Get().
    resource, err := client.TenancyV1alpha1().WorkspaceTypes().Get(ctx, wmwName, opts)
    //tenancyv1alpha1.Delete()

    // delete WMW
    // kubectl "${kubectl_flags[@]}" delete workspaces.tenancy.kcp.io "$wmw_name"

    fmt.Println(resource)

    return nil
//    return errors.New("rm wmw err")
}