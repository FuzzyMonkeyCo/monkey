package pkg

import (
	"fmt"
)

func newStatusError(expectedCode int, got string) error {
	return fmt.Errorf("expected status %d but got '%v'", expectedCode, got)
}
