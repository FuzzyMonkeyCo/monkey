package starlarktruth

import (
	"fmt"
	"strconv"
	"strings"

	"go.starlark.net/starlark"
)

var _ fmt.Stringer = (*duplicateCounter)(nil)

type duplicateCounter struct {
	m map[string]uint
	s []string
}

func newDuplicateCounter() *duplicateCounter {
	return &duplicateCounter{
		m: make(map[string]uint),
	}
}

func (c *duplicateCounter) empty() bool { return len(c.m) == 0 }

func (c *duplicateCounter) contains(v starlark.Value) bool {
	_, ok := c.m[v.String()]
	return ok
}

func (c *duplicateCounter) increment(v starlark.Value) {
	vv := v.String()
	if _, ok := c.m[vv]; !ok {
		c.m[vv] = 0
		c.s = append(c.s, vv)
	}
	c.m[vv] += 1
}

func (c *duplicateCounter) decrement(v starlark.Value) {
	vv := v.String()
	if count, ok := c.m[vv]; ok {
		if count != 1 {
			c.m[vv] -= 1
			return
		}
		delete(c.m, vv)
		if sz := len(c.s); sz != 1 {
			s := make([]string, 0, len(c.s)-1)
			for _, vvv := range c.s {
				if vvv != vv {
					s = append(s, vvv)
				}
			}
			c.s = s
		} else {
			c.s = nil
		}
	}
}

func (c *duplicateCounter) String() string {
	var b strings.Builder
	first := true
	// b.WriteString("[")
	for _, vv := range c.s {
		if !first {
			b.WriteString(", ")
		}
		first = false

		b.WriteString(vv)
		if count := c.m[vv]; count != 1 {
			b.WriteString(" [")
			b.WriteString(strconv.FormatUint(uint64(count), 10))
			b.WriteString(" copies]")
		}
	}
	// b.WriteString("]")
	return b.String()
}
