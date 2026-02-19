package plugin

import "fmt"

// FSMActionStart is a conventional action name for the Plugin.Start transition.
// Plugins are not required to use this name.
const FSMActionStart = "START"

// TransitionKey uniquely identifies an FSM edge by the current plugin state and the
// action being applied. An empty FromState represents the pre-initialised condition
// (plugin not yet started).
type TransitionKey struct {
	FromState string
	Action    string
}

// TransitionOutcome is the result of a valid FSM edge.
// NextTaskState of "" means the task-level state does not change on this transition.
type TransitionOutcome struct {
	NextPluginState string
	NextTaskState   State
}

// PluginFSM is a declarative, table-driven finite state machine for plugin state
// transitions. It is generic and plugin-agnostic; callers supply the full transition
// table at construction time.
//
// Usage:
//
//	fsm := NewPluginFSM(map[TransitionKey]TransitionOutcome{
//	    {"", FSMActionStart}:      {"INITIALISED", InProgress},
//	    {"INITIALISED", "SUBMIT"}: {"SUBMITTED", Completed},
//	})
//
//	outcome, err := fsm.Transition(currentPluginState, action)
type PluginFSM struct {
	transitions map[TransitionKey]TransitionOutcome
}

// NewPluginFSM returns a PluginFSM backed by the provided transition table.
// The table must be fully specified before calling NewPluginFSM; it is not safe
// to mutate after construction.
func NewPluginFSM(table map[TransitionKey]TransitionOutcome) *PluginFSM {
	return &PluginFSM{transitions: table}
}

// Transition looks up and returns the outcome for (currentState, action).
// Returns an error if the edge is not defined in the table.
func (f *PluginFSM) Transition(currentState, action string) (TransitionOutcome, error) {
	outcome, ok := f.transitions[TransitionKey{FromState: currentState, Action: action}]
	if !ok {
		return TransitionOutcome{}, fmt.Errorf(
			"fsm: action %q is not permitted in plugin state %q",
			action, currentState,
		)
	}
	return outcome, nil
}

// CanTransition reports whether action is a legal transition from currentState.
func (f *PluginFSM) CanTransition(currentState, action string) bool {
	_, ok := f.transitions[TransitionKey{FromState: currentState, Action: action}]
	return ok
}
