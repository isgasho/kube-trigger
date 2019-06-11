package e2e

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	appv1alpha1 "github.com/caitong93/kube-trigger/pkg/apis/app/v1alpha1"
	"github.com/caitong93/kube-trigger/pkg/trigger"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testDeployment(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	// get global framework variables
	f := framework.Global
	// wait for memcached-operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "kube-trigger", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Create workload
	dp := makeBusyboxDeployment(namespace)
	if _, err := f.KubeClient.AppsV1().Deployments(namespace).Create(dp); err != nil {
		t.Fatal(err)
	}
	dpName := dp.Name
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, dpName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Create Config
	cm := makeConfigmap(namespace)
	if _, err := f.KubeClient.CoreV1().ConfigMaps(namespace).Create(cm); err != nil {
		t.Fatal(err)
	}

	// Create CR
	cr := &appv1alpha1.TriggerRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: namespace,
		},
		Spec: appv1alpha1.TriggerRuleSpec{
			Sources: []appv1alpha1.Source{
				{
					ObjectRef: v1.ObjectReference{
						Kind:      "ConfigMap",
						Namespace: namespace,
						Name:      cm.Name,
					},
				},
			},
			Actions: []appv1alpha1.Action{
				{

					UpdatePodTemplate: &appv1alpha1.ActionUpdatePodTemplate{
						ObjectRef: v1.ObjectReference{
							Kind:      "Deployment",
							Namespace: namespace,
							Name:      dp.Name,
						},
					},
				},
			},
		},
	}
	if err := f.Client.Create(context.TODO(), cr, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		t.Fatal(err)
	}

	dp1, err := f.KubeClient.AppsV1().Deployments(namespace).Get(dp.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	randData := fmt.Sprint(time.Now().UnixNano())
	cm.Data[randData] = randData
	if _, err := f.KubeClient.CoreV1().ConfigMaps(namespace).Update(cm); err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)
	dp2, err := f.KubeClient.AppsV1().Deployments(namespace).Get(dp.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	labelKey := trigger.GetRecordKey(cr.Name, cr.Namespace)

	if !checkAnnotationUpdated(dp1.Spec.Template.Annotations, dp2.Spec.Template.Annotations, labelKey) {
		t.Errorf("Workload have not been updated")
		fmt.Println(mustToJSON(dp2))
	}

	// Wait a while to make sure workload will not be updated
	time.Sleep(15 * time.Second)
	dp3, err := f.KubeClient.AppsV1().Deployments(namespace).Get(dp.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(dp3.Spec.Template, dp2.Spec.Template) {
		t.Error("Workload should not be updated")
	}
}
