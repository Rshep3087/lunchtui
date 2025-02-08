package main

// loadingState is a map of keys to boolean values
// to determine if a key is in a loading state
type loadingState map[string]bool

func newLoadingState(keys ...string) loadingState {
	l := make(loadingState, len(keys))
	for _, k := range keys {
		l[k] = false
	}
	return l
}

// set sets the key in the loading state
func (l loadingState) set(key string) {
	l[key] = true
}

// unset unsets the key in the loading state
func (l loadingState) unset(key string) {
	l[key] = false
}

// allLoaded returns true if all keys are loaded
func (l loadingState) allLoaded() (bool, string) {
	for k, v := range l {
		if !v {
			return false, k
		}
	}

	return true, ""
}
