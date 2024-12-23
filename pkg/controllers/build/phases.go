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

package build

import (
	"context"

	buildv1 "github.com/forge-build/forge/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/util/conditions"
)

func (r *BuildReconciler) reconcilePhase(_ context.Context, build *buildv1.Build) {
	preReconcilePhase := build.Status.GetTypedPhase()

	if build.Status.Phase == "" {
		build.Status.SetTypedPhase(buildv1.BuildPhasePending)
		return
	}

	if build.Spec.InfrastructureRef != nil && conditions.Has(build, buildv1.InfrastructureReadyCondition) {
		build.Status.SetTypedPhase(buildv1.BuildPhaseBuilding)
	}

	if build.Status.InfrastructureReady {
		build.Status.SetTypedPhase(buildv1.BuildPhaseBuilding)
	}

	if build.Status.FailureReason != nil || build.Status.FailureMessage != nil {
		build.Status.SetTypedPhase(buildv1.BuildPhaseFailed)
	}

	if !build.DeletionTimestamp.IsZero() {
		build.Status.SetTypedPhase(buildv1.BuildPhaseTerminating)
	}

	if build.Status.Ready {
		build.Status.SetTypedPhase(buildv1.BuildPhaseCompleted)
	}

	// Only record the event if the status has changed
	if preReconcilePhase != build.Status.GetTypedPhase() {
		// Failed clusters should get a Warning event
		if build.Status.GetTypedPhase() == buildv1.BuildPhaseFailed {
			r.recorder.Eventf(build, corev1.EventTypeWarning, string(build.Status.GetTypedPhase()), "Build %s is %s: %s", build.Name, string(build.Status.GetTypedPhase()), ptr.Deref(build.Status.FailureMessage, "unknown"))
		} else {
			r.recorder.Eventf(build, corev1.EventTypeNormal, string(build.Status.GetTypedPhase()), "Build %s is %s", build.Name, string(build.Status.GetTypedPhase()))
		}
	}
}
