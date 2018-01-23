package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type docsInvalidError struct {
	Errors string
}

func (e *docsInvalidError) Error() string {
	return e.Errors
}

func newDocsInvalidError(errors []byte) *docsInvalidError {
	start, end := "Validation errors:", "Documentation validation failed."
	var theErrors string
	var out bytes.Buffer
	if err := json.Indent(&out, errors, "", "  "); err != nil {
		theErrors = string(errors)
	}
	theErrors = out.String()

	return &docsInvalidError{start + "\n" + theErrors + "\n" + end}
}

func newStatusError(expectedCode int, got string) error {
	return fmt.Errorf("expected status %d but got '%v'", expectedCode, got)
}
