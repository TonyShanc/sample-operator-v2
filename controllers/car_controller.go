/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	samplev1 "github.com/tonyshanc/sample-operator-v2/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AtReconciler reconciles a At object
type AtReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sample.github.com,resources=ats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sample.github.com,resources=ats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sample.github.com,resources=ats/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the At object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *AtReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling At")

	// Fetch the At instance
	instance := &samplev1.At{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// If no phase set. default to pending (the initial phase):
	if instance.Status.Phase == "" {
		instance.Status.Phase = samplev1.PhasePending
	}

	// Now let's make the main case distinction: implementing
	// the state diagram PENDING -> RUNNING -> DONE
	switch instance.Status.Phase {
	case samplev1.PhasePending:
		reqLogger.Info("Phase: PENDING")
		// As long as we haven't executed the command yet, we need to check if
		// it's already time to act
		reqLogger.Info("Checking schedule", "Target", instance.Spec.Schedule)
		// Check if it is ready time to execute the command with a tolerance
		// of 2 seconds:
		d, err := timeUntilSchedule(instance.Spec.Schedule)
		if err != nil {
			reqLogger.Error(err, "Schedule parsing failure")
			return ctrl.Result{}, err
		}
		reqLogger.Info("Schedule parsing done", "Result diff", fmt.Sprintf("%v", d))
		if d > 0 {
			return ctrl.Result{RequeueAfter: d}, err
		}
		reqLogger.Info("It's time!", "Ready to execute", instance.Spec.Command)
		instance.Status.Phase = samplev1.PhaseRunning
	case samplev1.PhaseRunning:
		reqLogger.Info("Phase: RUNNING")
		pod := newPodForAt(instance)
		if err := controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		found := &corev1.Pod{}
		nsName := types.NamespacedName{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		}

		err := r.Get(ctx, nsName, found)
		if err != nil && errors.IsNotFound(err) {
			if err = r.Create(ctx, pod); err != nil {
				return ctrl.Result{}, err
			}
			reqLogger.Info("Pod has launched requeue after 3s", "name", pod.Name)
			// enqueue to check if it has finished.
			return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
		} else if err != nil {
			return ctrl.Result{}, err
		} else if found.Status.Phase == corev1.PodFailed || found.Status.Phase == corev1.PodSucceeded {
			reqLogger.Info("Container terminated", "reason", found.Status.Reason, "message", found.Status.Message)
			instance.Status.Phase = samplev1.PhaseDone
		} else {
			reqLogger.Info("！！！Pod has launched requeue after 3s", "name", pod.Name)
			// enqueue to check if it has finished.
			return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
		}

	case samplev1.PhaseDone:
		reqLogger.Info("Phase: DONE")
		return ctrl.Result{}, nil

	default:
		reqLogger.Info("NOP")
		return ctrl.Result{}, nil
	}

	// Update the instance Phase:
	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AtReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&samplev1.At{}).
		Complete(r)
}

func timeUntilSchedule(schedule string) (time.Duration, error) {
	now := time.Now().UTC()
	layout := "2006-01-02T15:04:06Z"
	s, err := time.Parse(layout, schedule)
	if err != nil {
		return time.Duration(0), err
	}
	return s.Sub(now), nil
}

func newPodForAt(at *samplev1.At) *corev1.Pod {
	labels := map[string]string{
		"app": at.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      at.Name + "pod",
			Namespace: at.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: strings.Split(at.Spec.Command, " "),
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
	}
}
