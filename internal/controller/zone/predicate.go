package zone

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// debugPredicate is a reject-all predicate that logs the event kind and
// useful object metadata for debugging watch behavior. It is used as the
// second branch of a predicate.Or so events rejected by the first predicate
// are logged.
type debugPredicate struct{}

// Create implements predicate.Predicate.
func (debugPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		logf.Log.V(1).Info("watch event", "kind", "create")
		return false
	}
	logf.Log.V(1).Info("watch event",
		"kind", "create",
		"namespace", e.Object.GetNamespace(),
		"name", e.Object.GetName(),
		"generation", e.Object.GetGeneration(),
		"resourceVersion", e.Object.GetResourceVersion(),
	)
	return false
}

// Update implements predicate.Predicate.
func (debugPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew == nil {
		logf.Log.V(1).Info("watch event", "kind", "update")
		return false
	}
	logFields := []any{
		"kind", "update",
		"namespace", e.ObjectNew.GetNamespace(),
		"name", e.ObjectNew.GetName(),
		"generation", e.ObjectNew.GetGeneration(),
		"resourceVersion", e.ObjectNew.GetResourceVersion(),
		"deletionTimestamp", e.ObjectNew.GetDeletionTimestamp(),
	}
	if e.ObjectOld != nil {
		logFields = append(logFields,
			"oldGeneration", e.ObjectOld.GetGeneration(),
			"oldResourceVersion", e.ObjectOld.GetResourceVersion(),
			"oldDeletionTimestamp", e.ObjectOld.GetDeletionTimestamp(),
		)
	}
	logf.Log.V(1).Info("watch event", logFields...)
	return false
}

// Delete implements predicate.Predicate.
func (debugPredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		logf.Log.V(1).Info("watch event", "kind", "delete")
		return false
	}
	logf.Log.V(1).Info("watch event",
		"kind", "delete",
		"namespace", e.Object.GetNamespace(),
		"name", e.Object.GetName(),
		"generation", e.Object.GetGeneration(),
		"resourceVersion", e.Object.GetResourceVersion(),
		"deletionTimestamp", e.Object.GetDeletionTimestamp(),
	)
	return false
}

// Generic implements predicate.Predicate.
func (debugPredicate) Generic(e event.GenericEvent) bool {
	if e.Object == nil {
		logf.Log.V(1).Info("watch event", "kind", "generic")
		return false
	}
	logf.Log.V(1).Info("watch event",
		"kind", "generic",
		"namespace", e.Object.GetNamespace(),
		"name", e.Object.GetName(),
		"generation", e.Object.GetGeneration(),
		"resourceVersion", e.Object.GetResourceVersion(),
		"deletionTimestamp", e.Object.GetDeletionTimestamp(),
	)
	return false
}

// Predicates returns the predicates used to filter watch events for DNSZone
// objects: only generation changes are reconciled. Events rejected by the
// generation predicate are logged by the debug predicate, which always
// returns false.
func Predicates() []predicate.Predicate {
	return []predicate.Predicate{
		predicate.Or(predicate.GenerationChangedPredicate{}, debugPredicate{}),
	}
}
