/*
Copyright 2024 The Forge Authors.

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

package controller

import (
	"context"
	"fmt"

	"github.com/forge-build/forge/pkg/util"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/cluster-api/util/patch"

	buildv1 "github.com/forge-build/forge/pkg/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	k8sapierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var podControlledByJobNotFoundErr = errors.New("pod for job not found")

// ShellJobController watches Kubernetes jobs and reports back to the Build
type ShellJobController struct {
	Logger logr.Logger
	client.Client
	Clientset *kubernetes.Clientset
	Namespace string

	patchHelper *patch.Helper
}

func (r *ShellJobController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}, builder.WithPredicates(
			ManagedByForgeProvisionerShell,
			InNamespace(r.Namespace),
			JobHasAnyCondition,
			HasBuildNameLabel,
			HasProvisionerIDLabel,
		)).
		Complete(r.reconcileJobs())
}

//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;patch;update

func (r *ShellJobController) reconcileJobs() reconcile.Func {
	return func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		job := &batchv1.Job{}
		err := r.Client.Get(ctx, req.NamespacedName, job)
		if err != nil {
			if k8sapierror.IsNotFound(err) {
				r.Logger.Info("Ignoring cached job that must have been deleted")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, fmt.Errorf("getting job from cache: %w", err)
		}

		if len(job.Status.Conditions) == 0 {
			r.Logger.Info("Ignoring Job without conditions")
			return ctrl.Result{}, nil
		}

		buildName := job.GetLabels()[buildv1.BuildNameLabel]
		buildNamespace := job.GetLabels()[buildv1.BuildNamespaceLabel]
		provisionerID := job.GetLabels()[buildv1.ProvisionerIDLabel]

		build := &buildv1.Build{}
		err = r.Client.Get(ctx, client.ObjectKey{Namespace: buildNamespace, Name: buildName}, build)
		if err != nil {
			if k8sapierror.IsNotFound(err) {
				r.Logger.Info("Ignoring cached job that must have been deleted")
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, fmt.Errorf("getting build from cache: %w", err)
		}
		r.patchHelper, err = patch.NewHelper(build, r.Client)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to create patch helper")
		}

		switch jobCondition := job.Status.Conditions[0].Type; jobCondition {
		case batchv1.JobComplete:
			err = r.processCompleteScanJob(ctx, job, build, provisionerID)
		case batchv1.JobFailed:
			err = r.processFailedScanJob(ctx, job, build, provisionerID)
		default:
			err = fmt.Errorf("unrecognized scan job condition: %v", jobCondition)
		}
		if err != nil {
			r.Logger.Error(err, "Failed processing job")
		}

		return ctrl.Result{}, err
	}
}

// processCompleteScanJob handles the completed scan jobs
// report back to the queue with saving appropriate cache
func (r *ShellJobController) processCompleteScanJob(ctx context.Context, job *batchv1.Job, build *buildv1.Build, provisionerID string) error {
	r.Logger.Info("Job complete", "build", build.Name, "provisionerID", provisionerID)

	// TODO think about how to handle the output of the shell job (providing logs)

	// Update Build Provisioner Status
	provisioner, err := util.GetProvisionerByID(build, provisionerID)
	if err != nil {
		return errors.Wrapf(err, "unable to find provisioner with id %s in the build %s", provisionerID, build.Name)
	}
	provisioner.Status = ptr.To(buildv1.ProvisionerStatusCompleted)

	if err := r.patchHelper.Patch(ctx, build); err != nil {
		r.Logger.Error(err, "failed to patch build")
	}
	r.Logger.Info("Job complete - Deleting complete shell job", "job", job.Name)
	return r.deleteJob(ctx, job)
}

// nolint:gocyclo
func (r *ShellJobController) processFailedScanJob(ctx context.Context, job *batchv1.Job, build *buildv1.Build, provisionerID string) error {
	r.Logger.Info("Job failed", "build", build, "provisionerID", provisionerID)

	statuses, err := r.GetTerminatedContainersStatusesByJob(ctx, job)
	if err != nil {
		r.Logger.Error(err, "Could not get terminated container statuses")
		return err
	}

	provisioner, err := util.GetProvisionerByID(build, provisionerID)
	if err != nil {
		return errors.Wrapf(err, "unable to find provisioner with id %s in the build %s", provisionerID, build.Name)
	}

	for container, status := range statuses {
		if status.ExitCode == 0 {
			continue
		}
		errorMsg := fmt.Sprintf("shelljob failed with reason: %s and message: %s", status.Reason, status.Message)
		r.Logger.Error(errors.New("shell job failed"), "shell failed with reason", "build", build, "provisionerID", provisionerID, "container", container, "errorMessage", errorMsg)
		provisioner.FailureReason = ptr.To(status.Reason)
		provisioner.FailureMessage = ptr.To(status.Message)
	}

	provisioner.Status = ptr.To(buildv1.ProvisionerStatusFailed)

	if err := r.patchHelper.Patch(ctx, build); err != nil {
		r.Logger.Error(err, "failed to patch build")
	}

	r.Logger.Info("Deleting failed scan job")
	return r.deleteJob(ctx, job)
}

func (r *ShellJobController) deleteJob(ctx context.Context, job *batchv1.Job) error {
	err := r.Client.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground))
	if err != nil {
		if k8sapierror.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting job: %w", err)
	}
	return nil
}

func (r *ShellJobController) GetTerminatedContainersStatusesByJob(ctx context.Context, job *batchv1.Job) (map[string]*corev1.ContainerStateTerminated, error) {
	pod, err := r.getPodByJob(ctx, job)
	if err != nil {
		if k8sapierror.IsNotFound(err) {
			r.Logger.Info("Cached job must have been deleted")
			return nil, err
		}
		if IsPodControlledByJobNotFound(err) {
			r.Logger.Info("Pod must have been deleted")
			err = r.deleteJob(ctx, job)
			return nil, err
		}

		return nil, fmt.Errorf("unknown issue: %w", err)
	}

	statuses := GetTerminatedContainersStatusesByPod(pod)
	return statuses, nil
}

func (r *ShellJobController) getPodByJob(ctx context.Context, job *batchv1.Job) (*corev1.Pod, error) {
	refreshedJob, err := r.Clientset.BatchV1().Jobs(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	podList, err := r.podListLookup(ctx, job.Namespace, refreshedJob)
	if err != nil {
		return nil, err
	}
	if podList != nil && len(podList.Items) > 0 {
		return &podList.Items[0], nil
	}
	return nil, nil
}

func (r *ShellJobController) podListLookup(ctx context.Context, namespace string, refreshedJob *batchv1.Job) (*corev1.PodList, error) {
	matchingLabelKey := "controller-uid"
	matchingLabelValue := refreshedJob.Spec.Selector.MatchLabels[matchingLabelKey]
	if len(matchingLabelValue) == 0 {
		matchingLabelKey = "batch.kubernetes.io/controller-uid" // for k8s v1.27.x and above
		matchingLabelValue = refreshedJob.Spec.Selector.MatchLabels[matchingLabelKey]
	}
	selector := fmt.Sprintf("%s=%s", matchingLabelKey, matchingLabelValue)
	return r.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
}

func GetTerminatedContainersStatusesByPod(pod *corev1.Pod) map[string]*corev1.ContainerStateTerminated {
	states := make(map[string]*corev1.ContainerStateTerminated)
	if pod == nil {
		return states
	}
	for _, status := range pod.Status.InitContainerStatuses {
		if status.State.Terminated == nil {
			continue
		}
		states[status.Name] = status.State.Terminated
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Terminated == nil {
			continue
		}
		states[status.Name] = status.State.Terminated
	}
	return states
}

func IsPodControlledByJobNotFound(err error) bool {
	return errors.Is(err, podControlledByJobNotFoundErr)
}
