package protoeval

// Env describes an environment within which an evaluation can take place.
// Instances of this type are not safe for concurrent use. Clone your
// environment instead.
type Env struct {
	// values is the general value storage for this environment.
	values map[string]interface{}

	// scope is the current scope
	scope scope

	// cyclesLeft is the number of cycles (an evaluation cost measure) left
	// before we abort an evaluation.
	cyclesLeft int
}

// shiftScopeByName returns a shallow copy of this environment with the
// scope shifted by the specified name.
func (e *Env) shiftScopeByName(name string) (*Env, error) {
	newenv := *e
	var err error
	newenv.scope, err = e.scope.ShiftByName(name)
	if err != nil {
		return nil, err
	}
	return &newenv, nil
}

// shiftScopeByIndex returns a shallow copy of this environment with the
// scope shifted by the specified index.
func (e *Env) shiftScopeByIndex(index uint32) (*Env, error) {
	newenv := *e
	var err error
	newenv.scope, err = e.scope.ShiftByIndex(index)
	if err != nil {
		return nil, err
	}
	return &newenv, nil
}

// shiftScopeByBoolKey returns a shallow copy of this environment with the
// scope shifted by the specified key.
func (e *Env) shiftScopeByBoolKey(key bool) (*Env, error) {
	newenv := *e
	var err error
	newenv.scope, err = e.scope.ShiftByBoolKey(key)
	if err != nil {
		return nil, err
	}
	return &newenv, nil
}

// shiftScopeByUintKey returns a shallow copy of this environment with the
// scope shifted by the specified key.
func (e *Env) shiftScopeByUintKey(key uint64) (*Env, error) {
	newenv := *e
	var err error
	newenv.scope, err = e.scope.ShiftByUintKey(key)
	if err != nil {
		return nil, err
	}
	return &newenv, nil
}

// shiftScopeByIntKey returns a shallow copy of this environment with the
// scope shifted by the specified key.
func (e *Env) shiftScopeByIntKey(key int64) (*Env, error) {
	newenv := *e
	var err error
	newenv.scope, err = e.scope.ShiftByIntKey(key)
	if err != nil {
		return nil, err
	}
	return &newenv, nil
}

// shiftScopeToParent returns a shallow copy of this environment with the
// scope shifted to the parent scope.
func (e *Env) shiftScopeToParent() (*Env, error) {
	newenv := *e
	var err error
	newenv.scope, err = e.scope.ShiftToParent()
	if err != nil {
		return nil, err
	}
	return &newenv, nil
}
