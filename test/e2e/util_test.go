package e2e

import (
	"encoding/json"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func mustToJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

func checkAnnotationUpdated(prev, cur map[string]string, key string) bool {
	if cur == nil {
		return false
	}
	if prev == nil {
		return true
	}
	prevVal, prevExist := prev[key]
	curVal, curExist := cur[key]
	if !curExist {
		return false
	}
	if !prevExist {
		return true
	}
	return prevVal != curVal
}

func int32ptr(a int32) *int32 {
	return &a
}

func makeConfigmap(ns string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cm",
			Namespace: ns,
		},
		Data: map[string]string{"init": ""},
	}
}

func makeBusyboxDeployment(ns string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "busybox",
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32ptr(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "busybox",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "busybox"},
					Annotations: map[string]string{"foo": "bar"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "busybox",
							Image:   "busybox",
							Command: []string{"tail"},
							Args:    []string{"-f", "/dev/null"},
						},
					},
				},
			},
		},
	}
}

func makeBusyboxStatefulSetService(ns string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "busybox",
			Namespace: ns,
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Selector: map[string]string{
				"app": "busybox",
			},
			Ports: []v1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}
}

func makeBusyboxStatefulSet(ns string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "busybox",
			Namespace: ns,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "busybox",
			Replicas:    int32ptr(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "busybox",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "busybox"},
					Annotations: map[string]string{"foo": "bar"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "busybox",
							Image:   "busybox",
							Command: []string{"tail"},
							Args:    []string{"-f", "/dev/null"},
						},
					},
				},
			},
		},
	}
}

func makeBusyboxDeamonset(ns string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "busybox",
			Namespace: ns,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "busybox",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "busybox"},
					Annotations: map[string]string{"foo": "bar"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "busybox",
							Image:   "busybox",
							Command: []string{"tail"},
							Args:    []string{"-f", "/dev/null"},
						},
					},
				},
			},
		},
	}
}

func waitForStatefulSet(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, replicas int, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		sts, err := kubeclient.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s statefulset\n", name)
				return false, nil
			}
			return false, err
		}

		if int(sts.Status.ReadyReplicas) == replicas {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s statefulset (%d/%d)\n", name, sts.Status.ReadyReplicas, replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Statefulset available (%d/%d)\n", replicas, replicas)
	return nil
}

func waitForDaemonSet(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	var replicas int
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ds, err := kubeclient.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s daemonset\n", name)
				return false, nil
			}
			return false, err
		}
		replicas = int(ds.Status.DesiredNumberScheduled)
		if int(ds.Status.NumberAvailable) == replicas {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s daemonset (%d/%d)\n", name, ds.Status.NumberAvailable, replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Statefulset available (%d/%d)\n", replicas, replicas)
	return nil
}
