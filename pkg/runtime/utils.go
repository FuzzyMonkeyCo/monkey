package runtime

import (
	"errors"
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

func legalName(s string) error {
	l := 0
	for _, c := range s {
		switch {
		case c > unicode.MaxASCII:
			return fmt.Errorf("string contains non-ASCII characters: %q", s)
		case !unicode.IsPrint(c):
			return fmt.Errorf("string contains non-printable characters: %q", s)
		case unicode.IsUpper(c):
			return fmt.Errorf("string contains upper case characters: %q", s)
		}
		l++
	}
	switch {
	case l == 0:
		return errors.New("empty strings are illegal")
	case l > 255:
		return fmt.Errorf("string must be shorter than 256 characters: %q", s)
	}
	return nil
}
