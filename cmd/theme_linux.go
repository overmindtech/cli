package cmd

import (
	"bytes"
	"os"
	"sync"
)

var isWslCache int // 0 = unset; 1 = WSL; 2 = not WSL
var isWslCacheMu sync.RWMutex

// IsConhost returns true if the current terminal is conhost. This indicates
// that it can't deal with multi-byte characters and requires special treatment.
// See https://github.com/overmindtech/cli/issues/388 for detailed analysis.
func IsConhost() bool {
	// shortcut this if we (probably) run in Windows Terminal (through WSL) or
	// on something that smells like a regular Linux terminal
	if os.Getenv("WT_SESSION") != "" {
		return false
	}

	isWslCacheMu.RLock()
	w := isWslCache
	isWslCacheMu.RUnlock()

	switch w {
	case 1:
		return true
	case 2:
		return false
	}

	// isWslCache has not yet been initialised, so we need to check if we are in WSL
	// since we don't know if we are in WSL, we need to check now
	isWslCacheMu.Lock()
	defer isWslCacheMu.Unlock()
	if w != 0 {
		// someone else raced the lock and has already decided
		return isWslCache == 1
	}

	// check if we run in WSL
	ver, err := os.ReadFile("/proc/version")
	if err == nil && bytes.Contains(ver, []byte("Microsoft")) {
		isWslCache = 1
		return true
	}

	// we can't access /proc/version or it does not contain Microsoft, we are _probably_ not in WSL
	isWslCache = 2
	return false
}
