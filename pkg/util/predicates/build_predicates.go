/*
Copyright 2020 The Kubernetes Authors.

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

// Package predicates implements predicate utilities.
package predicates

import (
	"fmt"

	buildv1 "github.com/forge-build/forge/pkg/api/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// BuildUpdateUnpaused returns a predicate that returns true for an update event when a build has Spec.Paused changed from true to false
// it also returns true if the resource provided is not a Build to allow for use with controller-runtime NewControllerManagedBy.
func BuildUpdateUnpaused(logger logr.Logger) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log := logger.WithValues("predicate", "ClusterUpdateUnpaused", "eventType", "update")

			oldCluster, ok := e.ObjectOld.(*buildv1.Build)
			if !ok {
				log.V(4).Info("Expected Build", "type", fmt.Sprintf("%T", e.ObjectOld))
				return false
			}
			log = log.WithValues("Build", klog.KObj(oldCluster))

			newCluster := e.ObjectNew.(*buildv1.Build)

			if oldCluster.Spec.Paused && !newCluster.Spec.Paused {
				log.V(4).Info("Cluster was unpaused, allowing further processing")
				return true
			}

			// This predicate always work in "or" with Paused predicates
			// so the logs are adjusted to not provide false negatives/verbosity al V<=5.
			log.V(6).Info("Cluster was not unpaused, blocking further processing")
			return false
		},
		CreateFunc:  func(event.CreateEvent) bool { return false },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}
