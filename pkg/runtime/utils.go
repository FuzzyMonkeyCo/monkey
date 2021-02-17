package runtime

import (
	"fmt"
	"unicode"
)

func printableASCIItmp(s string) (reserved bool, err error) {
	l := 0
	for i, c := range s {
		if i == 0 && unicode.IsUpper(c) {
			reserved = true
		}
		if !(c <= unicode.MaxASCII && unicode.IsPrint(c)) {
			err = fmt.Errorf("string contains non-ASCII or non-printable characters: %q", s)
			return
		}
		l++
	}
	if l > 255 {
		err = fmt.Errorf("string must be shorter than 256 characters: %q", s)
	}
	return
}
