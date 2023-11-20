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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned"

	ksclientset "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned"
)

// Get the mailbox name for a given SyncTarget name
func GetMailboxName(client *ksclientset.Clientset, ctx context.Context, syncTargetName string) (string, error) {
	// Get the SyncTarget
	syncTarget, err := client.EdgeV2alpha1().SyncTargets().Get(ctx, syncTargetName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// Mailbox name is CLUSTER_NAME + "-mb-" + UID
	mbClusterName := syncTarget.ObjectMeta.Annotations["kcp.io/cluster"]
	mbUID := string(syncTarget.ObjectMeta.UID)
	mbName := mbClusterName + "-mb-" + mbUID

	return mbName, nil
}

// Check if APIExport exists, returning a boolean for state
func CheckAPIExportExists(client *kcpclientset.Clientset, ctx context.Context, apiExportName string) (bool, error) {
	_, err := client.ApisV1alpha1().APIExports().Get(ctx, apiExportName, metav1.GetOptions{})
	if err == nil {
		// APIExport exists
		return true, nil
	} else if apierrors.IsNotFound(err) {
		// No APIExport
		return false, nil
	}
	// Some error other than a non-existant APIExport
	return false, err
}

// Check that Workspace exists, returning a boolean for state
func CheckWorkspaceExists(client *kcpclientset.Clientset, ctx context.Context, wsName string) (bool, error) {
	_, err := client.TenancyV1alpha1().Workspaces().Get(ctx, wsName, metav1.GetOptions{})
	if err == nil {
		// Workspace exists
		return true, nil
	} else if apierrors.IsNotFound(err) {
		// No Workspace
		return false, nil
	}
	// Some error other than a non-existant Workspace
	return false, err
}

// Check if mailbox exists, returning a boolean for state.
// We will give 3 chances to find the mailbox workspace, with 15 second waits between.
func CheckMailboxExists(client *kcpclientset.Clientset, ctx context.Context, mbName string) (bool, error) {
	exists, err := CheckWorkspaceExists(client, ctx, mbName)
	iter := 1
	for iter < 3 {
		if err == nil {
			// Got a result without error, return it
			return exists, err
		}
		time.Sleep(time.Second * 15)
		exists, err = CheckMailboxExists(client, ctx, mbName)
		iter += 1
	}
	// It has been 3 iterations, return whatever result we got regardless of error
	return exists, err
}