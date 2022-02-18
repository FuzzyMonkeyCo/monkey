package modeler

import (
	"errors"
	"fmt"
	"strings"
)

// See (v2) https://github.com/go-yaml/yaml/blob/7649d4548cb53a614db133b2a8ac1f31859dda8c/readerc.go#L357
// See (v3) https://github.com/go-yaml/yaml/blob/496545a6307b2a7d7a710fd516e5e16e8ab62dbc/readerc.go#L379
// TODO: whence https://github.com/go-yaml/yaml/issues/737#issuecomment-847519537
func isControl(r rune) rune {
	// Check if the character is in the allowed range:
	switch {

	//      #x9 | #xA | #xD | [#x20-#x7E]               (8 bit)
	case r == 0x09:
	case r == 0x0A:
	case r == 0x0D:
	case r >= 0x20 && r <= 0x7E:

	//      | #x85 | [#xA0-#xD7FF] | [#xE000-#xFFFD]    (16 bit)
	case r == 0x85:
	case r >= 0xA0 && r <= 0xD7FF:
	case r >= 0xE000 && r <= 0xFFFD:

	//      | [#x10000-#x10FFFF]                        (32 bit)
	case r >= 0x10000 && r <= 0x10FFFF:

	default:
		return -1
	}
	return r
}

// FindControlCharacters finds control characters that annoy YAML.
func FindControlCharacters(str string) (err error) {
	found := make(map[rune]struct{})
	for _, r := range str {
		if isControl(r) < 0 {
			found[r] = struct{}{}
		}
	}

	if len(found) != 0 {
		var msg strings.Builder
		msg.WriteString("found control characters:")
		for r := range found {
			msg.WriteString(fmt.Sprintf(" %U", r))
		}
		err = errors.New(msg.String())
	}
	return
}

// StripControlCharacters removes control characters that annoy YAML.
func StripControlCharacters(str string) string {
	return strings.Map(isControl, str)
}
