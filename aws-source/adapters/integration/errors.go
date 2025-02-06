package integration

import "fmt"

type NotFoundError struct {
	ResourceName string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("Resource not found: %s", e.ResourceName)
}

func NewNotFoundError(resourceName string) NotFoundError {
	return NotFoundError{ResourceName: resourceName}
}
