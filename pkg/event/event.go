/*
Copyright 2018 The Kubernetes Authors.

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

package event

import "github.com/samuelkuklis/controller-runtime/pkg/client"

// CreateEvent is an event where a Kubernetes object was created.  CreateEvent should be generated
// by a source.Source and transformed into a reconcile.Request by an handler.EventHandler.
type CreateEvent struct {
	// Object is the object from the event
	Object client.Object
}

// UpdateEvent is an event where a Kubernetes object was updated.  UpdateEvent should be generated
// by a source.Source and transformed into a reconcile.Request by an handler.EventHandler.
type UpdateEvent struct {
	// ObjectOld is the object from the event
	ObjectOld client.Object

	// ObjectNew is the object from the event
	ObjectNew client.Object
}

// DeleteEvent is an event where a Kubernetes object was deleted.  DeleteEvent should be generated
// by a source.Source and transformed into a reconcile.Request by an handler.EventHandler.
type DeleteEvent struct {
	// Object is the object from the event
	Object client.Object

	// DeleteStateUnknown is true if the Delete event was missed but we identified the object
	// as having been deleted.
	DeleteStateUnknown bool
}

// GenericEvent is an event where the operation type is unknown (e.g. polling or event originating outside the cluster).
// GenericEvent should be generated by a source.Source and transformed into a reconcile.Request by an
// handler.EventHandler.
type GenericEvent struct {
	// Object is the object from the event
	Object client.Object
}
