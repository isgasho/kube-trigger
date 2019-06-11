package e2e

import (
	"testing"
	"time"

	"github.com/caitong93/kube-trigger/pkg/apis"
	appv1alpha1 "github.com/caitong93/kube-trigger/pkg/apis/app/v1alpha1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestKubeTrigger(t *testing.T) {
	list := &appv1alpha1.TriggerRuleList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, list)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	t.Run("group", func(t *testing.T) {
		t.Run("Deployment", testDeployment)
		t.Run("StatefulSet", testStatefulSet)
		t.Run("DaemonSet", testDaemonSet)
	})
}
