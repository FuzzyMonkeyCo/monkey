package starlarktruth

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

var _ fmt.Stringer = (*duplicateCounter)(nil)

// duplicateCounter is a synchronized collection of counters for tracking duplicates.
//
// The count values may be modified only through Increment() and Decrement(),
// which increment and decrement by 1 (only). If a count ever becomes 0, the item
// is immediately expunged from the dictionary. Counts can never be negative;
// attempting to Decrement an absent key has no effect.
//
// Order is preserved so that error messages containing expected values match.
//
// Supports counting values based on their (starlark.Value).String() representation.
type duplicateCounter struct {
	m map[string]uint
	s []string
	d uint
}

func newDuplicateCounter() *duplicateCounter {
	return &duplicateCounter{
		m: make(map[string]uint),
	}
}

// HasDupes indicates whether there are values that appears > 1 times.
func (c *duplicateCounter) HasDupes() bool { return c.d != 0 }

func (c *duplicateCounter) Empty() bool { return len(c.m) == 0 }

func (c *duplicateCounter) Contains(v starlark.Value) bool {
	return c.contains(v.String())
}

func (c *duplicateCounter) contains(v string) bool {
	_, ok := c.m[v]
	return ok
}

// Increment increments a count by 1. Inserts the item if not present.
func (c *duplicateCounter) Increment(v starlark.Value) {
	c.increment(v.String())
}

func (c *duplicateCounter) increment(v string) {
	if _, ok := c.m[v]; !ok {
		c.m[v] = 0
		c.s = append(c.s, v)
	}
	c.m[v]++
	if c.m[v] == 2 {
		c.d++
	}
}

// Decrement decrements a count by 1. Expunges the item if the count is 0.
// If the item is not present, has no effect.
func (c *duplicateCounter) Decrement(v starlark.Value) {
	c.decrement(v.String())
}

func (c *duplicateCounter) decrement(v string) {
	if count, ok := c.m[v]; ok {
		if count != 1 {
			c.m[v]--
			if c.m[v] == 1 {
				c.d--
			}
			return
		}
		delete(c.m, v)
		if sz := len(c.s); sz != 1 {
			s := make([]string, 0, len(c.s)-1)
			for _, vv := range c.s {
				if vv != v {
					s = append(s, vv)
				}
			}
			c.s = s
		} else {
			c.s = nil
		}
	}
}

// Returns the string representation of the duplicate counts.
//
// Items occurring more than once are accompanied by their count.
// Otherwise the count is implied to be 1.
//
// For example, if the internal dict is {2: 1, 3: 4, 'abc': 1}, this returns
// the string "[{2, 3 [4 copies], 'abc'}]".
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
			b.WriteString(fmt.Sprintf("%d", count))
			b.WriteString(" copies]")
		}
	}
	// b.WriteString("]")
	return b.String()
}

// Dupes shows only items whose count > 1.
func (c *duplicateCounter) Dupes() string {
	var b strings.Builder
	first := true
	for _, vv := range c.s {
		if count := c.m[vv]; count != 1 {
			if !first {
				b.WriteString(", ")
			}
			first = false

			b.WriteString(vv)
			b.WriteString(" [")
			b.WriteString(fmt.Sprintf("%d", count))
			b.WriteString(" copies]")
		}
	}
	return b.String()
}
