package discovery

import (
	"fmt"
	"sync"
)

// GetListMutex A modified version of a RWMutex. Many get locks can be held but
// only one List lock. A waiting List lock (even if it hasn't been locked, just
// if someone is waiting) blocks all other get locks until it unlocks.
//
// The intended usage of this is that it will allow an adapter which is trying to
// process many queries at once, to process a LIST query before any GET
// queries, since it's likely that once LIST has been run, subsequent GET
// queries will be able to be served from cache
type GetListMutex struct {
	mutexMap map[string]*sync.RWMutex
	mapLock  sync.Mutex
}

// GetLock Gets a lock that can be held by an unlimited number of goroutines,
// these locks are only blocked by ListLocks. A type and scope must be
// provided since a Get in one type (or scope) should not be blocked by a List
// in another
func (g *GetListMutex) GetLock(scope string, typ string) {
	g.mutexFor(scope, typ).RLock()
}

// GetUnlock Unlocks the GetLock. This must be called once for each GetLock
// otherwise it will be impossible to ever obtain a ListLock
func (g *GetListMutex) GetUnlock(scope string, typ string) {
	g.mutexFor(scope, typ).RUnlock()
}

// ListLock An exclusive lock. Ensure that all GetLocks have been unlocked and
// stops any more from being obtained. Provide a type and scope to ensure that
// the lock is only help for that type and scope combination rather than
// locking the whole engine
func (g *GetListMutex) ListLock(scope string, typ string) {
	g.mutexFor(scope, typ).Lock()
}

// ListUnlock Unlocks a ListLock
func (g *GetListMutex) ListUnlock(scope string, typ string) {
	g.mutexFor(scope, typ).Unlock()
}

// mutexFor Returns the relevant RWMutex for a given scope and type, creating
// and storing a new one if needed
func (g *GetListMutex) mutexFor(scope string, typ string) *sync.RWMutex {
	var mutex *sync.RWMutex
	var ok bool

	keyName := g.keyName(scope, typ)

	g.mapLock.Lock()
	defer g.mapLock.Unlock()

	// Create the map if needed
	if g.mutexMap == nil {
		g.mutexMap = make(map[string]*sync.RWMutex)
	}

	// Get the mutex from storage
	mutex, ok = g.mutexMap[keyName]

	// If the mutex wasn't found for this key, create a new one
	if !ok {
		mutex = &sync.RWMutex{}
		g.mutexMap[keyName] = mutex
	}

	return mutex
}

// keyName Returns the name of the key for a given scope and type combo for
// use with the mutexMap
func (g *GetListMutex) keyName(scope string, typ string) string {
	return fmt.Sprintf("%v.%v", scope, typ)
}
