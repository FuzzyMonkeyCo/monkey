package runtime

import (
	"fmt"
	"unicode"
)

func printableASCII(s string) error {
	l := 0
	for _, c := range s {
		if !(c <= unicode.MaxASCII && unicode.IsPrint(c)) {
			return fmt.Errorf("string contains non-ASCII or non-printable characters: %q", s)
		}
		l++
	}
	if l > 255 {
		return fmt.Errorf("string must be shorter than 256 characters: %q", s)
	}
	return nil
}
