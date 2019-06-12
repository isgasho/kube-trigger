package triggerrule

import (
	"context"
	"time"

	appv1alpha1 "github.com/caitong93/kube-trigger/pkg/apis/app/v1alpha1"
	"github.com/caitong93/kube-trigger/pkg/trigger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_triggerrule")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new TriggerRule Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileTriggerRule{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// Issue: cannot get kind information using o.Object.GetObjectKind() due to https://github.com/kubernetes/client-go/issues/541
// To workaround this here pass kind as a parameter.
func enqueTriggerRuleForConfig(c client.Client, kind string) handler.ToRequestsFunc {
	return func(o handler.MapObject) []reconcile.Request {
		rules := &appv1alpha1.TriggerRuleList{}
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if err := c.List(ctx, &client.ListOptions{}, rules); err != nil {
			log.Error(err, "err list rules")
			return nil
		}

		var reqs []reconcile.Request
		// Find TriggerRules which references the object
		for _, item := range rules.Items {
			for _, src := range item.Spec.Sources {
				ref := src.ObjectRef
				if ref.Kind == kind && ref.Namespace == o.Meta.GetNamespace() && ref.Name == o.Meta.GetName() {
					reqs = append(reqs, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: item.Namespace,
							Name:      item.Name,
						},
					})
					break
				}
			}
		}
		return reqs
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("triggerrule-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource TriggerRule
	err = c.Watch(&source.Kind{Type: &appv1alpha1.TriggerRule{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to ConfigMaps and requeue the related TriggerRule
	if err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: enqueTriggerRuleForConfig(mgr.GetClient(), "ConfigMap")}); err != nil {
		return err
	}

	// Watch for changes to Secrets and requeue the related TriggerRule
	if err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: enqueTriggerRuleForConfig(mgr.GetClient(), "Secret")}); err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileTriggerRule implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTriggerRule{}

// ReconcileTriggerRule reconciles a TriggerRule object
type ReconcileTriggerRule struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a TriggerRule object and makes changes based on the state read
// and what is in the TriggerRule.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTriggerRule) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling TriggerRule")

	// Fetch the TriggerRule instance
	instance := &appv1alpha1.TriggerRule{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	trigger.Add(request.NamespacedName, instance)

	return reconcile.Result{}, nil
}
