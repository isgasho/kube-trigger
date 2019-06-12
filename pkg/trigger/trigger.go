package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	appv1alpha1 "github.com/caitong93/kube-trigger/pkg/apis/app/v1alpha1"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var (
	// Use a global instance so we can close trigger when the process exit.
	// TODO: find a better way to avoid use global instance.
	global Trigger
)

// Init muse be called before using global instance.
func Init(client kubernetes.Interface, logger logr.Logger) {
	if global != nil {
		panic("Trigger should not be init more than once")
	}
	global = New(client, logger)
	global.Start()
}

// Add add an event to queue of trigger to be processed latter.
func Add(key types.NamespacedName, rule *appv1alpha1.TriggerRule) {
	global.Add(key, rule)
}

// Stop stops the trigger.
func Stop() {
	global.Stop()
}

// Trigger will process events triggered by updates of sources.
type Trigger interface {
	// Add add an event to queue of trigger  to be processed latter. Duplicated events with the same key
	// will be merged to only keep the lastest one.
	Add(key types.NamespacedName, rule *appv1alpha1.TriggerRule)
	// Start starts running the trigger.
	Start()
	// Stop stops the trigger.
	Stop()
}

var _ Trigger = &DefaultTrigger{}

// DefaultTrigger ...
type DefaultTrigger struct {
	ctx    context.Context
	cancel func()
	logger logr.Logger
	client kubernetes.Interface
	mu     sync.Mutex
	events map[types.NamespacedName]*appv1alpha1.TriggerRule
}

// New creates a new trigger
func New(client kubernetes.Interface, logger logr.Logger) Trigger {
	ctx, cancel := context.WithCancel(context.Background())
	return &DefaultTrigger{
		ctx:    ctx,
		cancel: cancel,
		client: client,
		logger: logger,
		events: make(map[types.NamespacedName]*appv1alpha1.TriggerRule),
	}
}

// Start implements Trigger.
func (t *DefaultTrigger) Start() {
	go t.process()
}

// Stop implements Trigger.
func (t *DefaultTrigger) Stop() {
	t.cancel()
}

// Add implements Trigger.
func (t *DefaultTrigger) Add(key types.NamespacedName, rule *appv1alpha1.TriggerRule) {
	t.mu.Lock()
	defer t.mu.Unlock()

	v, exist := t.events[key]
	if exist && v.ResourceVersion > rule.ResourceVersion {
		return
	}
	t.events[key] = rule
}

// TODO:
// 1. add multiple workers
// 2. use sync.Cond
func (t *DefaultTrigger) process() {
	for {
		select {
		case <-t.ctx.Done():
			// TODO: process the remaining items
			t.logger.Info("Quit.")
			return
		default:
		}

		t.mu.Lock()
		if len(t.events) > 0 {
			var (
				key  types.NamespacedName
				rule *appv1alpha1.TriggerRule
			)
			for key, rule = range t.events {
				break
			}
			delete(t.events, key)
			t.mu.Unlock()

			t.logger.Info("Process", "key", key)
			if err := t.do(t.ctx, rule); err != nil {
				t.logger.Error(err, "Action failed", "rule", rule)
			}
		} else {
			t.mu.Unlock()
			time.Sleep(1 * time.Second)
		}
	}
}

func (t *DefaultTrigger) do(ctx context.Context, rule *appv1alpha1.TriggerRule) error {
	// Get last state of sources, and set ResourceVersion
	var g errgroup.Group
	for i := range rule.Spec.Sources {
		src := &rule.Spec.Sources[i]
		ref := &src.ObjectRef
		g.Go(func() error {
			switch ref.Kind {
			case "ConfigMap":
				cm, err := t.client.CoreV1().ConfigMaps(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
				if err != nil {
					return fmt.Errorf("err get configmap: %v", err)
				}
				ref.ResourceVersion = cm.ResourceVersion
			case "Secret":
				sc, err := t.client.CoreV1().Secrets(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
				if err != nil {
					return fmt.Errorf("err get secret: %v", err)
				}
				ref.ResourceVersion = sc.ResourceVersion
			default:
				return fmt.Errorf("unsupported source kind %v", ref.Kind)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("err check sources: %v", err)
	}

	var actionG errgroup.Group
	for i := range rule.Spec.Actions {
		actionG.Go(func() error {
			// rule will only be read in following process, it's ok to not make a copy
			// TODO: retry
			return t.action(ctx, rule, &rule.Spec.Actions[i])
		})
	}
	if err := actionG.Wait(); err != nil {
		return fmt.Errorf("err execute actions: %v", err)
	}

	return nil
}

func (t *DefaultTrigger) action(ctx context.Context, rule *appv1alpha1.TriggerRule, action *appv1alpha1.Action) error {
	if action.UpdatePodTemplate != nil {
		return t.updatePodTemplate(ctx, rule, action)
	} else {
		return fmt.Errorf("no action to execute")
	}
}

func (t *DefaultTrigger) updatePodTemplate(ctx context.Context, rule *appv1alpha1.TriggerRule, action *appv1alpha1.Action) error {
	ref := action.UpdatePodTemplate.ObjectRef
	annotationKey := GetRecordKey(rule.Name, rule.Namespace)

	// TODO: refactor latter to reduce redundancy
	switch ref.Kind {
	case "Deployment":
		d, err := t.client.AppsV1().Deployments(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("err get deployment: %v", err)
		}

		rec, err := t.generateNewRecord(rule, d.Spec.Template.Annotations, annotationKey)
		if err != nil {
			return fmt.Errorf("err generate record: %v", err)
		}
		if rec == nil {
			return nil
		}

		pt, err := generatePatch(rec, annotationKey, d.Spec.Template.Annotations == nil)
		if err != nil {
			return fmt.Errorf("err generate patch: %v", err)
		}

		t.logger.Info("Generate patch", "patch", string(pt))
		if _, err := t.client.AppsV1().Deployments(d.Namespace).Patch(d.Name, types.JSONPatchType, pt); err != nil {
			iErr := err.(*errors.StatusError)
			return fmt.Errorf("err patch workload: %#v", iErr)
		}
		return nil
	case "StatefulSet":
		sts, err := t.client.AppsV1().StatefulSets(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("err get statefulset: %v", err)
		}

		rec, err := t.generateNewRecord(rule, sts.Spec.Template.Annotations, annotationKey)
		if err != nil {
			return fmt.Errorf("err generate record: %v", err)
		}
		if rec == nil {
			return nil
		}

		pt, err := generatePatch(rec, annotationKey, sts.Spec.Template.Annotations == nil)
		if err != nil {
			return fmt.Errorf("err generate patch: %v", err)
		}

		t.logger.Info("Generate patch", "patch", string(pt))
		if _, err := t.client.AppsV1().StatefulSets(sts.Namespace).Patch(sts.Name, types.JSONPatchType, pt); err != nil {
			iErr := err.(*errors.StatusError)
			return fmt.Errorf("err patch workload: %#v", iErr)
		}
		return nil
	case "DaemonSet":
		ds, err := t.client.AppsV1().DaemonSets(ref.Namespace).Get(ref.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("err get daemonset: %v", err)
		}

		rec, err := t.generateNewRecord(rule, ds.Spec.Template.Annotations, annotationKey)
		if err != nil {
			return fmt.Errorf("err generate record: %v", err)
		}
		if rec == nil {
			return nil
		}

		pt, err := generatePatch(rec, annotationKey, ds.Spec.Template.Annotations == nil)
		if err != nil {
			return fmt.Errorf("err generate patch: %v", err)
		}

		t.logger.Info("Generate patch", "patch", string(pt))
		if _, err := t.client.AppsV1().DaemonSets(ds.Namespace).Patch(ds.Name, types.JSONPatchType, pt); err != nil {
			iErr := err.(*errors.StatusError)
			return fmt.Errorf("err patch workload: %#v", iErr)
		}
		return nil
	default:
		return fmt.Errorf("unsupported workload kind %v", ref.Kind)
	}
}

// generateNewRecord return nil Record if sources are not changed.
func (t *DefaultTrigger) generateNewRecord(rule *appv1alpha1.TriggerRule, annotations map[string]string, key string) (*Record, error) {
	rec, err := decodeRecordFromAnnotaion(annotations, key)
	if err != nil {
		return nil, err
	}

	// Verify record data is correct
	// Record is invalid in two cases:
	// 1. annotation is modified by others
	// 2. TriggerRule updated
	invalidRec := false
	if rec != nil {
		if len(rec.Sources) != len(rule.Spec.Sources) {
			invalidRec = true
		}
		for i := range rule.Spec.Sources {
			a := rule.Spec.Sources[i].ObjectRef
			b := rec.Sources[i]
			if a.Name == b.Name && a.Namespace == b.Namespace && a.Kind == b.Kind {
				continue
			}
			invalidRec = true
			break
		}
		if invalidRec {
			t.logger.Error(fmt.Errorf("%#v", rec), "record data is invalid", "rule", rule)
		}
	}

	var newRec *Record
	if rec == nil || invalidRec {
		newRec = &Record{LastUpdateTime: time.Now().UnixNano()}
		for _, src := range rule.Spec.Sources {
			newRec.Sources = append(newRec.Sources, Source{
				Name:            src.ObjectRef.Name,
				Namespace:       src.ObjectRef.Namespace,
				Kind:            src.ObjectRef.Kind,
				ResourceVersion: src.ObjectRef.ResourceVersion,
			})
		}
	} else {
		// TODO: check configmap change by compare hash of content
		update := false
		for i, src := range rule.Spec.Sources {
			a := src.ObjectRef
			b := rec.Sources[i]
			if a.ResourceVersion > b.ResourceVersion {
				update = true
				rec.Sources[i].ResourceVersion = a.ResourceVersion
			}
		}
		if !update {
			return nil, nil
		}
		newRec = rec
	}
	return newRec, nil
}

func decodeRecordFromAnnotaion(annotations map[string]string, key string) (*Record, error) {
	if len(annotations) == 0 {
		return nil, nil
	}

	v, exist := annotations[key]
	if !exist {
		return nil, nil
	}

	ret := &Record{}
	if err := json.Unmarshal([]byte(v), ret); err != nil {
		return nil, fmt.Errorf("err decode annotaion <%v, %v>: %v", key, v, err)
	}
	return ret, nil
}
