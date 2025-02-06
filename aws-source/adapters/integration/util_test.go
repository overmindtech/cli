package integration

import (
	"os"
	"testing"
)

func Test_testID(t *testing.T) {
	t.Run("test id is given via env var", func(t *testing.T) {
		err := os.Setenv("INTEGRATION_TEST_ID", "test-id")
		if err != nil {
			t.Error(err)
		}
		defer func() {
			err := os.Unsetenv("INTEGRATION_TEST_ID")
			if err != nil {
				t.Error(err)
			}
		}()

		if got := TestID(); got != "test-id" {
			t.Errorf("TestID() = %v, want %v", got, "test-id")
		}
	})

	t.Run("test id is not given via env var - defaults to host name", func(t *testing.T) {
		err := os.Unsetenv("INTEGRATION_TEST_ID")
		if err != nil {
			t.Error(err)
		}

		if got := TestID(); got == "" {
			t.Errorf("TestID() = %v, want not empty", got)
		}
	})
}
