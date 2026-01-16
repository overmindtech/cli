package mocks

import (
	"go.uber.org/mock/gomock"
)

// Type aliases and helper functions to match the expected mock names in tests.
// These allow tests to use the shorter names while the actual mocks implement
// the concrete interfaces.

// MockVirtualMachinesPager is a type alias for MockVirtualMachinesPagerInterface
// to match the expected name in tests.
type MockVirtualMachinesPager = MockVirtualMachinesPagerInterface

// NewMockVirtualMachinesPager creates a new mock instance of VirtualMachinesPager.
// This is a convenience function that matches the expected name in tests.
// Returns the concrete mock type so tests can call .EXPECT() on it.
func NewMockVirtualMachinesPager(ctrl *gomock.Controller) *MockVirtualMachinesPager {
	return NewMockVirtualMachinesPagerInterface(ctrl)
}

// MockStorageAccountsPager is a type alias for MockStorageAccountsPagerInterface
// to match the expected name in tests.
type MockStorageAccountsPager = MockStorageAccountsPagerInterface

// NewMockStorageAccountsPager creates a new mock instance of StorageAccountsPager.
// This is a convenience function that matches the expected name in tests.
// Returns the concrete mock type so tests can call .EXPECT() on it.
func NewMockStorageAccountsPager(ctrl *gomock.Controller) *MockStorageAccountsPager {
	return NewMockStorageAccountsPagerInterface(ctrl)
}
