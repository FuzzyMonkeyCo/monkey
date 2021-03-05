package starlarktruth

import "testing"

func TestCannotCompareToNone(t *testing.T) {
	p := "It is illegal to compare using ."
	testEach(t, map[string]error{
		`that(5).is_at_least(None)`:     newInvalidAssertion(p + "is_at_least(None)"),
		`that(5).is_at_most(None)`:      newInvalidAssertion(p + "is_at_most(None)"),
		`that(5).is_greater_than(None)`: newInvalidAssertion(p + "is_greater_than(None)"),
		`that(5).is_less_than(None)`:    newInvalidAssertion(p + "is_less_than(None)"),
	})
}

func TestNone(t *testing.T) {
	testEach(t, map[string]error{
		`that(None).is_none()`:      nil,
		`that(None).is_not_none()`:  fail(`None`, `is not None`),
		`that("abc").is_not_none()`: nil,
		`that("abc").is_none()`:     fail(abc, `is None`),
	})
}
