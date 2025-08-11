package dynamic

import (
	"fmt"
)

type PermissionError struct {
	URL string
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission denied: %s", e.URL)
}
