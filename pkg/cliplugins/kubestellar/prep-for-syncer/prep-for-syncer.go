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

package plugin

import (
	"context"
	"fmt"
	//"io"
	//"os"

	//corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/kubernetes/scheme"
	//"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/remotecommand"

	ksclientset "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned"
)


func GetMailboxName(client *ksclientset.Clientset, ctx context.Context, syncTargetName string) (string, error) {
	// Delete the SyncTarget
	syncTarget, err := client.EdgeV2alpha1().SyncTargets().Get(ctx, syncTargetName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	fmt.Println(syncTarget)

	return "", nil
}