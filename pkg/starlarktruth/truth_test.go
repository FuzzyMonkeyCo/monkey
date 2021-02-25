package starlarktruth

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

type asWhat int

const (
	asFunc asWhat = iota
	asModule
)

const abc = `"abc"` // Please linter

func helper(t *testing.T, as asWhat, program string) (starlark.StringDict, error) {
	// Enabled so they can be tested
	resolve.AllowFloat = true
	resolve.AllowSet = true
	resolve.AllowLambda = true

	predeclared := starlark.StringDict{}
	if as == asModule {
		NewModule(predeclared)
	} else {
		starlark.Universe["that"] = starlark.NewBuiltin("that", That)
	}

	thread := &starlark.Thread{
		Name: t.Name(),
		Print: func(_ *starlark.Thread, msg string) {
			t.Logf("--> %s", msg)
		},
		Load: func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
			return nil, errors.New("load() disabled")
		},
	}

	script := strings.Join([]string{
		`dfltCmp = ` + cmpSrc,
		`someCmp = lambda a, b: dfltCmp(b, a)`,
		program,
	}, "\n")

	d, err := starlark.ExecFile(thread, t.Name()+".star", script, predeclared)
	if err != nil {
		return nil, err
	}
	if err := Close(thread); err != nil {
		return nil, err
	}
	return d, nil
}

func testEach(t *testing.T, m map[string]error, asSlice ...asWhat) {
	as := asFunc
	for _, as = range asSlice {
	}
	for code, expectedErr := range m {
		t.Run(code, func(t *testing.T) {
			globals, err := helper(t, as, code)
			delete(globals, "dfltCmp")
			delete(globals, "someCmp")
			delete(globals, "fortytwo")
			require.Empty(t, globals)
			if expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.EqualError(t, err, expectedErr.Error())
				require.True(t, errors.As(err, &expectedErr))
				require.IsType(t, expectedErr, err)
			}
		})
	}
}

func fail(value, expected string, suffixes ...string) error {
	var suffix string
	switch len(suffixes) {
	case 0:
	case 1:
		suffix = suffixes[0]
	default:
		panic(`There must be only one suffix`)
	}
	msg := "Not true that <" + value + "> " + expected + "." + suffix
	return newTruthAssertion(msg)
}

func TestClosedness(t *testing.T) {
	testEach(t, map[string]error{
		`assert.that(True)`:                          IntegrityError("TestClosedness/assert.that(True).star:3:12"),
		`assert.that(True).is_true()`:                nil,
		`assert.that(True).named("eh")`:              IntegrityError(`TestClosedness/assert.that(True).named("eh").star:3:12`),
		`assert.that(True).named("eh").is_true()`:    nil,
		`assert.that(assert.that).is_not_callable()`: fail(`built-in method assert of assert value`, `is not callable`),
	}, asModule)
}

func TestAsValue(t *testing.T) {
	testEach(t, map[string]error{
		`
fortytwo = that(42)
fortytwo.is_equal_to(42.0)
fortytwo.is_not_callable()
fortytwo.is_at_least(42)
`: nil,
	})
}

func TestTrue(t *testing.T) {
	testEach(t, map[string]error{
		`that(True).is_true()`:  nil,
		`that(True).is_false()`: fail("True", "is False"),
	})
}

func TestFalse(t *testing.T) {
	testEach(t, map[string]error{
		`that(False).is_false()`: nil,
		`that(False).is_true()`:  fail("False", "is True"),
	})
}

func TestTruthyThings(t *testing.T) {
	values := []string{
		`1`,
		`True`,
		`2.5`,
		`"Hi"`,
		`[3]`,
		`{4: "four"}`,
		`("my", "tuple")`,
		`set([5])`,
		`-1`,
	}
	m := make(map[string]error, 4*len(values))
	for _, v := range values {
		m[`that(`+v+`).is_truthy()`] = nil
		m[`that(`+v+`).is_falsy()`] = fail(v, "is falsy")
		m[`that(`+v+`).is_false()`] = fail(v, "is False")
		if v != `True` {
			m[`that(`+v+`).is_true()`] = fail(v, "is True",
				" However, it is truthy. Did you mean to call .is_truthy() instead?")
		}
	}
	testEach(t, m)
}

func TestFalsyThings(t *testing.T) {
	values := []string{
		`None`,
		`False`,
		`0`,
		`0.0`,
		`""`,
		`()`, // tuple
		`[]`,
		`{}`,
		`set()`,
	}
	m := make(map[string]error, 4*len(values))
	for _, v := range values {
		vv := v
		if v == `set()` {
			vv = `set([])`
		}
		m[`that(`+v+`).is_falsy()`] = nil
		m[`that(`+v+`).is_truthy()`] = fail(vv, "is truthy")
		m[`that(`+v+`).is_true()`] = fail(vv, "is True")
		if v != `False` {
			m[`that(`+v+`).is_false()`] = fail(vv, "is False",
				" However, it is falsy. Did you mean to call .is_falsy() instead?")
		}
	}
	testEach(t, m)
}

func TestIsAtLeast(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_at_least(3)`: nil,
		`that(5).is_at_least(5)`: nil,
		`that(5).is_at_least(8)`: fail("5", "is at least <8>"),
	})
}

func TestIsAtMost(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_at_most(5)`: nil,
		`that(5).is_at_most(8)`: nil,
		`that(5).is_at_most(3)`: fail("5", "is at most <3>"),
	})
}

func TestIsGreaterThan(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_greater_than(3)`: nil,
		`that(5).is_greater_than(5)`: fail("5", "is greater than <5>"),
		`that(5).is_greater_than(8)`: fail("5", "is greater than <8>"),
	})
}

func TestIsLessThan(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_less_than(8)`: nil,
		`that(5).is_less_than(5)`: fail("5", "is less than <5>"),
		`that(5).is_less_than(3)`: fail("5", "is less than <3>"),
	})
}

func TestCannotCompareToNone(t *testing.T) {
	p := "It is illegal to compare using ."
	testEach(t, map[string]error{
		`that(5).is_at_least(None)`:     newInvalidAssertion(p + "is_at_least(None)"),
		`that(5).is_at_most(None)`:      newInvalidAssertion(p + "is_at_most(None)"),
		`that(5).is_greater_than(None)`: newInvalidAssertion(p + "is_greater_than(None)"),
		`that(5).is_less_than(None)`:    newInvalidAssertion(p + "is_less_than(None)"),
	})
}

func TestIsEqualTo(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_equal_to(5)`: nil,
		`that(5).is_equal_to(3)`: fail("5", "is equal to <3>"),
		`that({1:2,3:4}).is_equal_to([1,2,3,4])`: fail(`{1: 2, 3: 4}`,
			"is equal to <[1, 2, 3, 4]>"),
	})
}

func TestIsEqualToFailsOnFloatsAsWellAsWithFormattedRepresentations(t *testing.T) {
	testEach(t, map[string]error{
		`that(0.3).is_equal_to(0.1+0.2)`: fail("0.3", "is equal to <0.30000000000000004>"),
		`that(0.1+0.2).is_equal_to(0.3)`: fail("0.30000000000000004", "is equal to <0.3>"),
	})
}

func TestIsNotEqualTo(t *testing.T) {
	testEach(t, map[string]error{
		`that(5).is_not_equal_to(3)`: nil,
		`that(5).is_not_equal_to(5)`: fail("5", "is not equal to <5>"),
	})
}

func TestSequenceIsEqualToUsesContainsExactlyElementsInPlusInOrder(t *testing.T) {
	testEach(t, map[string]error{
		`that((3,5,[])).is_equal_to((3, 5, []))`: nil,
		`that((3,5,[])).is_equal_to(([],3,5))`: fail("(3, 5, [])",
			"contains exactly these elements in order <([], 3, 5)>"),
		`that((3,5,[])).is_equal_to((3,5,[],9))`: fail("(3, 5, [])",
			"contains exactly <(3, 5, [], 9)>. It is missing <9>"),
		`that((3,5,[])).is_equal_to((9,3,5,[],10))`: fail("(3, 5, [])",
			"contains exactly <(9, 3, 5, [], 10)>. It is missing <9, 10>"),
		`that((3,5,[])).is_equal_to((3,5))`: fail("(3, 5, [])",
			"contains exactly <(3, 5)>. It has unexpected items <[]>"),
		`that((3,5,[])).is_equal_to(([],3))`: fail("(3, 5, [])",
			"contains exactly <([], 3)>. It has unexpected items <5>"),
		`that((3,5,[])).is_equal_to((3,))`: fail("(3, 5, [])",
			"contains exactly <(3,)>. It has unexpected items <5, []>"),
		`that((3,5,[])).is_equal_to((4,4,3,[],5))`: fail("(3, 5, [])",
			"contains exactly <(4, 4, 3, [], 5)>. It is missing <4 [2 copies]>"),
		`that((3,5,[])).is_equal_to((4,4))`: fail("(3, 5, [])",
			"contains exactly <(4, 4)>. It is missing <4 [2 copies]> and has unexpected items <3, 5, []>"),
		`that((3,5,[])).is_equal_to((3,5,9))`: fail("(3, 5, [])",
			"contains exactly <(3, 5, 9)>. It is missing <9> and has unexpected items <[]>"),
		`that((3,5,[])).is_equal_to(())`: fail("(3, 5, [])", "is empty"),
	})
}

func TestSetIsEqualToUsesContainsExactlyElementsIn(t *testing.T) {
	s := `that(set([3, 5, 8]))`
	testEach(t, map[string]error{
		s + `.is_equal_to(set([3, 5, 8]))`: nil,
		s + `.is_equal_to(set([8, 3, 5]))`: nil,
		s + `.is_equal_to(set([3, 5, 8, 9]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5, 8, 9])>. It is missing <9>"),
		s + `.is_equal_to(set([9, 3, 5, 8, 10]))`: fail("set([3, 5, 8])",
			"contains exactly <set([9, 3, 5, 8, 10])>. It is missing <9, 10>"),
		s + `.is_equal_to(set([3, 5]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5])>. It has unexpected items <8>"),
		s + `.is_equal_to(set([8, 3]))`: fail("set([3, 5, 8])",
			"contains exactly <set([8, 3])>. It has unexpected items <5>"),
		s + `.is_equal_to(set([3]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3])>. It has unexpected items <5, 8>"),
		s + `.is_equal_to(set([4]))`: fail("set([3, 5, 8])",
			"contains exactly <set([4])>. It is missing <4> and has unexpected items <3, 5, 8>"),
		s + `.is_equal_to(set([3, 5, 9]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5, 9])>. It is missing <9> and has unexpected items <8>"),
		s + `.is_equal_to(set([]))`: fail("set([3, 5, 8])", "is empty"),
	})
}

func TestSequenceIsEqualToComparedWithNonIterables(t *testing.T) {
	testEach(t, map[string]error{
		`that((3, 5, [])).is_equal_to(3)`: fail("(3, 5, [])", "is equal to <3>"),
	})
}

func TestSetIsEqualToComparedWithNonIterables(t *testing.T) {
	testEach(t, map[string]error{
		`that(set([3, 5, 8])).is_equal_to(3)`: fail("set([3, 5, 8])", "is equal to <3>"),
	})
}

func TestOrderedDictIsEqualToUsesContainsExactlyItemsInPlusInOrder(t *testing.T) {
	d1 := `((2, "two"), (4, "four"))`
	d2 := `((2, "two"), (4, "four"))`
	d3 := `((4, "four"), (2, "two"))`
	d4 := `((2, "two"), (4, "for"))`
	d5 := `((2, "two"), (4, "four"), (5, "five"))`
	s := `that(` + d1 + `).is_equal_to(`
	testEach(t, map[string]error{
		s + d2 + `)`: nil,
		s + d3 + `)`: fail(d1, "contains exactly these elements in order <"+d3+">"),

		s + `((2, "two"),))`: fail(d1,
			`contains exactly <((2, "two"),)>. It has unexpected items <(4, "four")>`),

		s + d4 + `)`: fail(d1,
			"contains exactly <"+d4+`>. It is missing <(4, "for")> and has unexpected items <(4, "four")>`),
		s + d5 + `)`: fail(d1,
			"contains exactly <"+d5+`>. It is missing <(5, "five")>`),
	})
}

func TestDictIsEqualToUsesContainsExactlyItemsIn(t *testing.T) {
	d := `{2: "two", 4: "four"}`
	dd := `{2: "two", 4: "for"}`
	ddd := `{2: "two", 4: "four", 5: "five"}`
	dBis := `items([(2, "two"), (4, "four")])`
	s := `that(` + d + `).is_equal_to(`
	testEach(t, map[string]error{
		s + d + `)`: nil,

		s + `{2: "two"})`: fail(dBis,
			`contains exactly <((2, "two"),)>. It has unexpected items <(4, "four")>`,
			warnContainsExactlySingleIterable),

		s + dd + `)`: fail(dBis,
			`contains exactly <((2, "two"), (4, "for"))>. It is missing <(4, "for")> and has unexpected items <(4, "four")>`),
		s + ddd + `)`: fail(dBis,
			`contains exactly <((2, "two"), (4, "four"), (5, "five"))>. It is missing <(5, "five")>`),
		s + `{})`: fail(`items([(2, "two"), (4, "four")])`, "is empty"),
	})
}

func TestIsEqualToComparedWithNonDictionary(t *testing.T) {
	testEach(t, map[string]error{
		`that({2:"two",4:"four"}).is_equal_to(3)`: fail(
			`{2: "two", 4: "four"}`,
			`is equal to <3>`,
		),
	})
}

func TestNamedMultilineString(t *testing.T) {
	s := `that("line1\nline2").named("some-name")`
	testEach(t, map[string]error{
		s + `.is_equal_to("line1\nline2")`: nil,
		s + `.is_equal_to("")`: newTruthAssertion(
			`Not true that actual some-name is equal to <"">.`),
		s + `.is_equal_to("line1\nline2\n")`: newTruthAssertion(
			`Not true that actual some-name is equal to expected, found diff:
*** Expected
--- Actual
***************
*** 1,3 ****
  line1
  line2
- 
--- 1,2 ----
.`),
	})
}

func TestIsEqualToRaisesErrorWithVerboseDiff(t *testing.T) {
	testEach(t, map[string]error{
		`that("line1\nline2\nline3\nline4\nline5\n") \
         .is_equal_to("line1\nline2\nline4\nline6\n")`: newTruthAssertion(
			`Not true that actual is equal to expected, found diff:
*** Expected
--- Actual
***************
*** 1,5 ****
  line1
  line2
  line4
! line6
  
--- 1,6 ----
  line1
  line2
+ line3
  line4
! line5
  
.`),
	})
}

func TestContainsExactly(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_exactly(` + x + `)`
	}
	testEach(t, map[string]error{
		`that(` + ss + `).contains_exactly_in_order(3, 5, [])`: nil,
		`that(` + ss + `).contains_exactly(3, 5, [])`:          nil,
		`that(` + ss + `).contains_exactly([], 3, 5)`:          nil,
		`that(` + ss + `).contains_exactly_in_order([], 3, 5)`: fail(ss,
			"contains exactly these elements in order <([], 3, 5)>"),

		s(`3, 5, [], 9`): fail(ss,
			`contains exactly <(3, 5, [], 9)>. It is missing <9>`),
		s(`9, 3, 5, [], 10`): fail(ss,
			`contains exactly <(9, 3, 5, [], 10)>. It is missing <9, 10>`),
		s(`3, 5`): fail(ss,
			`contains exactly <(3, 5)>. It has unexpected items <[]>`),
		s(`[], 3`): fail(ss,
			`contains exactly <([], 3)>. It has unexpected items <5>`),
		s(`3`): fail(ss,
			`contains exactly <(3,)>. It has unexpected items <5, []>`),
		s(`4, 4`): fail(ss,
			`contains exactly <(4, 4)>. It is missing <4 [2 copies]> and has unexpected items <3, 5, []>`),
		s(`3, 5, 9`): fail(ss,
			`contains exactly <(3, 5, 9)>. It is missing <9> and has unexpected items <[]>`),
		s(`(3, 5, [])`): fail(ss,
			`contains exactly <((3, 5, []),)>. It is missing <(3, 5, [])> and has unexpected items <3, 5, []>`,
			warnContainsExactlySingleIterable),
		s(``): fail(ss, "is empty"),
	})
}

func TestContainsExactlyDoesNotWarnIfSingleStringNotContained(t *testing.T) {
	s := `.contains_exactly("abc")`
	testEach(t, map[string]error{
		`that(())` + s:      fail(`()`, `contains exactly <("abc",)>. It is missing <"abc">`),
		`that([])` + s:      fail(`[]`, `contains exactly <("abc",)>. It is missing <"abc">`),
		`that({})` + s:      errMustBeEqualNumberOfKVPairs(1),
		`that("")` + s:      fail(`""`, `contains exactly <("abc",)>. It is missing <"abc">`),
		`that(set([]))` + s: fail(`set([])`, `contains exactly <("abc",)>. It is missing <"abc">`),
	})
}

func TestContainsExactlyEmptyContainer(t *testing.T) {
	s := func(x string) string {
		return `that(` + x + `).contains_exactly(3)`
	}
	testEach(t, map[string]error{
		s(`()`): fail(`()`, `contains exactly <(3,)>. It is missing <3>`),
		s(`[]`): fail(`[]`, `contains exactly <(3,)>. It is missing <3>`),
		s(`{}`): errMustBeEqualNumberOfKVPairs(1),
		s(`""`): fail(`""`, `contains exactly <(3,)>. It is missing <3>`),
		//FIXME: Not true that <''> contains exactly <(3,)>. It is missing <[3]>. warnContainsExactlySingleIterable
		s(`set([])`): fail(`set([])`, `contains exactly <(3,)>. It is missing <3>`),
	})
}

func TestContainsExactlyNothing(t *testing.T) {
	s := func(x string) string {
		return `that(` + x + `).contains_exactly()`
	}
	testEach(t, map[string]error{
		s(`()`):      nil,
		s(`[]`):      nil,
		s(`{}`):      nil,
		s(`""`):      nil,
		s(`set([])`): nil,
	})
}

func TestContainsExactlyElementsIn(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_exactly_elements_in(` + x + `)`
	}
	testEach(t, map[string]error{
		`that(` + ss + `).contains_exactly_elements_in_order_in((3, 5, []))`: nil,
		`that(` + ss + `).contains_exactly_elements_in(([], 3, 5))`:          nil,
		`that(` + ss + `).contains_exactly_elements_in_order_in(([], 3, 5))`: fail(ss,
			"contains exactly these elements in order <([], 3, 5)>"),

		s(`(3, 5, [], 9)`):     fail(ss, `contains exactly <(3, 5, [], 9)>. It is missing <9>`),
		s(`(9, 3, 5, [], 10)`): fail(ss, `contains exactly <(9, 3, 5, [], 10)>. It is missing <9, 10>`),
		s(`(3, 5)`):            fail(ss, `contains exactly <(3, 5)>. It has unexpected items <[]>`),
		s(`([], 3)`):           fail(ss, `contains exactly <([], 3)>. It has unexpected items <5>`),
		s(`(3,)`):              fail(ss, `contains exactly <(3,)>. It has unexpected items <5, []>`),
		s(`(4, 4)`):            fail(ss, `contains exactly <(4, 4)>. It is missing <4 [2 copies]> and has unexpected items <3, 5, []>`),
		s(`(3, 5, 9)`):         fail(ss, `contains exactly <(3, 5, 9)>. It is missing <9> and has unexpected items <[]>`),
		s(`()`):                fail(ss, `is empty`),
	})
}

func TestContainsExactlyElementsInEmptyContainer(t *testing.T) {
	testEach(t, map[string]error{
		`that(()).contains_exactly_elements_in(())`: nil,
		`that(()).contains_exactly_elements_in((3,))`: fail(`()`,
			`contains exactly <(3,)>. It is missing <3>`),
	})
}

func TestContainsExactlyTargetingOrderedDict(t *testing.T) {
	ss := `((2, "two"), (4, "four"))`
	s := `that(` + ss + `).contains_exactly(`
	testEach(t, map[string]error{
		`that(` + ss + `).contains_exactly_in_order((2, "two"), (4, "four"))`: nil,
		`that(` + ss + `).contains_exactly((2, "two"), (4, "four"))`:          nil,
		`that(` + ss + `).contains_exactly((4, "four"), (2, "two"))`:          nil,
		`that(` + ss + `).contains_exactly_in_order((4, "four"), (2, "two"))`: fail(ss,
			`contains exactly these elements in order <((4, "four"), (2, "two"))>`),

		s + `2, "two")`: fail(ss,
			`contains exactly <(2, "two")>. It is missing <2, "two"> and has unexpected items <(2, "two"), (4, "four")>`),

		s + `2, "two", 4, "for")`: fail(ss,
			`contains exactly <(2, "two", 4, "for")>. It is missing <2, "two", 4, "for"> and has unexpected items <(2, "two"), (4, "four")>`),

		s + `2, "two", 4, "four", 5, "five")`: fail(ss,
			`contains exactly <(2, "two", 4, "four", 5, "five")>. It is missing <2, "two", 4, "four", 5, "five"> and has unexpected items <(2, "two"), (4, "four")>`),
	})
}

func TestContainsExactlyPassingOddNumberOfArgs(t *testing.T) {
	testEach(t, map[string]error{
		`that({}).contains_exactly("key1", "value1", "key2")`: errMustBeEqualNumberOfKVPairs(3),
	})
}

func TestContainsExactlyItemsIn(t *testing.T) {
	s := func(x string) string {
		return `that({2: "two", 4: "four"}).contains_exactly_items_in(` + x + `)`
	}
	ss := `items([(2, "two"), (4, "four")])`
	testEach(t, map[string]error{
		// NOTE: not ported over from pytruth
		// as Starlark's Dict is not ordered (uses insertion order)
		// d2 = collections.OrderedDict(((2, 'two'), (4, 'four')))
		// d3 = collections.OrderedDict(((4, 'four'), (2, 'two')))
		// self.assertIsInstance(s.ContainsExactlyItemsIn(d2), truth._InOrder)
		// self.assertIsInstance(s.ContainsExactlyItemsIn(d3), truth._NotInOrder)
		// FIXME: still test for unhandled

		s(`{2: "two"}`): fail(ss,
			`contains exactly <((2, "two"),)>. It has unexpected items <(4, "four")>`,
			warnContainsExactlySingleIterable),

		s(`{2: "two", 4: "for"}`): fail(ss,
			`contains exactly <((2, "two"), (4, "for"))>. It is missing <(4, "for")> and has unexpected items <(4, "four")>`),

		s(`{2: "two", 4: "four", 5: "five"}`): fail(ss,
			`contains exactly <((2, "two"), (4, "four"), (5, "five"))>. It is missing <(5, "five")>`),
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

func TestIsIn(t *testing.T) {
	s := func(x string) string {
		return `that(3).is_in(` + x + `)`
	}
	testEach(t, map[string]error{
		`that("a").is_in("abc")`: nil,
		`that("d").is_in("abc")`: fail(`"d"`, `is equal to any of <"abc">`),
		s(`(3,)`):                nil,
		s(`(3, 5)`):              nil,
		s(`(1, 3, 5)`):           nil,
		s(`{3: "three"}`):        nil,
		s(`set([3, 5])`):         nil,
		s(`()`):                  fail(`3`, `is equal to any of <()>`),
		s(`(2,)`):                fail(`3`, `is equal to any of <(2,)>`),
	})
}

func TestIsNotIn(t *testing.T) {
	s := func(x string) string {
		return `that(3).is_not_in(` + x + `)`
	}
	testEach(t, map[string]error{
		`that("a").is_not_in("abc")`: fail(`"a"`, `is not in "abc". It was found at index 0`),
		`that("d").is_not_in("abc")`: nil,
		s(`(5,)`):                    nil,
		s(`set([5])`):                nil,
		s(`("3",)`):                  nil,
		s(`(3,)`):                    fail(`3`, `is not in (3,). It was found at index 0`),
		s(`(1, 3)`):                  fail(`3`, `is not in (1, 3). It was found at index 1`),
		s(`set([3])`):                fail(`3`, `is not in set([3])`),
	})
}

func TestIsAnyOf(t *testing.T) {
	s := func(x string) string {
		return `that(3).is_any_of(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):       nil,
		s(`3, 5`):    nil,
		s(`1, 3, 5`): nil,
		s(``):        fail(`3`, `is equal to any of <()>`),
		s(`2`):       fail(`3`, `is equal to any of <(2,)>`),
	})
}

func TestIsNoneOf(t *testing.T) {
	s := func(x string) string {
		return `that(3).is_none_of(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`5`):    nil,
		s(`"3"`):  nil,
		s(`3`):    fail(`3`, `is not in (3,). It was found at index 0`),
		s(`1, 3`): fail(`3`, `is not in (1, 3). It was found at index 1`),
	})
}

func TestHasAttribute(t *testing.T) {
	s := func(x string) string {
		return `that("my str").has_attribute(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"elems"`):    nil,
		s(`"index"`):    nil,
		s(`"isdigit"`):  nil,
		s(`""`):         fail(`"my str"`, `has attribute <"">`),
		s(`"ermagerd"`): fail(`"my str"`, `has attribute <"ermagerd">`),
	})
}

func TestDoesNotHaveAttribute(t *testing.T) {
	s := func(x string) string {
		return `that({1: ()}).does_not_have_attribute(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"other_attribute"`): nil,
		s(`""`):                nil,
		s(`"keys"`):            fail(`{1: ()}`, `does not have attribute <"keys">`),
		s(`"values"`):          fail(`{1: ()}`, `does not have attribute <"values">`),
		s(`"setdefault"`):      fail(`{1: ()}`, `does not have attribute <"setdefault">`),
	})
}

func TestIsCallable(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_callable()`
	}
	testEach(t, map[string]error{
		s(`lambda x: x`):    nil,
		s(`"str".endswith`): nil,
		s(`that`):           nil,
		s(`None`):           fail(`None`, `is callable`),
		s(abc):              fail(abc, `is callable`),
	})
}

func TestIsNotCallable(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_not_callable()`
	}
	testEach(t, map[string]error{
		s(`None`):           nil,
		s(abc):              nil,
		s(`lambda x: x`):    fail(`function lambda`, `is not callable`),
		s(`"str".endswith`): fail(`built-in method endswith of string value`, `is not callable`),
		s(`that`):           fail(`built-in function that`, `is not callable`),
	})
}

func TestHasSize(t *testing.T) {
	s := func(x string) string {
		return `that((2, 5, 8)).has_size(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):  nil,
		s(`-1`): fail(`(2, 5, 8)`, `has a size of <-1>. It is <3>`),
		s(`2`):  fail(`(2, 5, 8)`, `has a size of <2>. It is <3>`),
	})
}

func TestIsEmpty(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_empty()`
	}
	testEach(t, map[string]error{
		s(`()`):       nil,
		s(`[]`):       nil,
		s(`{}`):       nil,
		s(`set([])`):  nil,
		s(`""`):       nil,
		s(`(3,)`):     fail(`(3,)`, `is empty`),
		s(`[4]`):      fail(`[4]`, `is empty`),
		s(`{5: 6}`):   fail(`{5: 6}`, `is empty`),
		s(`set([7])`): fail(`set([7])`, `is empty`),
		s(`"height"`): fail(`"height"`, `is empty`),
	})
}

func TestIsNotEmpty(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_not_empty()`
	}
	testEach(t, map[string]error{
		s(`(3,)`):     nil,
		s(`[4]`):      nil,
		s(`{5: 6}`):   nil,
		s(`set([7])`): nil,
		s(`"height"`): nil,
		s(`()`):       fail(`()`, `is not empty`),
		s(`[]`):       fail(`[]`, `is not empty`),
		s(`{}`):       fail(`{}`, `is not empty`),
		s(`set([])`):  fail(`set([])`, `is not empty`),
		s(`""`):       fail(`""`, `is not empty`),
	})
}

func TestContains(t *testing.T) {
	s := func(x string) string {
		return `that((2, 5, [])).contains(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`2`):   nil,
		s(`5`):   nil,
		s(`[]`):  nil,
		s(`3`):   newTruthAssertion(`<(2, 5, [])> should have contained 3`),
		s(`"2"`): newTruthAssertion(`<(2, 5, [])> should have contained "2"`),
		s(`{}`):  newTruthAssertion(`<(2, 5, [])> should have contained {}`),
	})
}

func TestDoesNotContain(t *testing.T) {
	s := func(x string) string {
		return `that((2, 5, [])).does_not_contain(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):   nil,
		s(`"2"`): nil,
		s(`{}`):  nil,
		s(`2`):   newTruthAssertion(`<(2, 5, [])> should not have contained 2`),
		s(`5`):   newTruthAssertion(`<(2, 5, [])> should not have contained 5`),
		s(`[]`):  newTruthAssertion(`<(2, 5, [])> should not have contained []`),
	})
}

func TestContainsNoDuplicates(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).contains_no_duplicates()`
	}
	testEach(t, map[string]error{
		s(`()`):           nil,
		s(abc):            nil,
		s(`(2,)`):         nil,
		s(`(2, 5)`):       nil,
		s(`{2: 2}`):       nil,
		s(`set([2])`):     nil,
		s(`"aaa"`):        newTruthAssertion(`<"aaa"> has the following duplicates: <"a" [3 copies]>`),
		s(`(3, 2, 5, 2)`): newTruthAssertion(`<(3, 2, 5, 2)> has the following duplicates: <2 [2 copies]>`),
		s(`"abcabc"`): newTruthAssertion(
			`<"abcabc"> has the following duplicates:` +
				` <"a" [2 copies], "b" [2 copies], "c" [2 copies]>`),
	})
}

func TestContainsAllIn(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_all_in(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`()`):                                nil,
		`that(` + ss + `).contains_all_in(())`: nil,
		`that(` + ss + `).contains_all_in_order_in(())`:         nil,
		`that(` + ss + `).contains_all_in((3,))`:                nil,
		`that(` + ss + `).contains_all_in_order_in((3,))`:       nil,
		`that(` + ss + `).contains_all_in((3, []))`:             nil,
		`that(` + ss + `).contains_all_in_order_in((3, []))`:    nil,
		`that(` + ss + `).contains_all_in((3, 5, []))`:          nil,
		`that(` + ss + `).contains_all_in_order_in((3, 5, []))`: nil,
		`that(` + ss + `).contains_all_in(([], 5, 3))`:          nil,
		`that(` + ss + `).contains_all_in_order_in(([], 5, 3))`: fail(ss,
			`contains all elements in order <([], 5, 3)>`),
		s(`(2, 3)`):    fail(ss, "contains all elements in <(2, 3)>. It is missing <2>"),
		s(`(2, 3, 6)`): fail(ss, "contains all elements in <(2, 3, 6)>. It is missing <2, 6>"),
	})
}

func TestContainsAllOf(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_all_of(` + x + `)`
	}
	testEach(t, map[string]error{
		`that(` + ss + `).contains_all_of()`:                  nil,
		`that(` + ss + `).contains_all_of_in_order()`:         nil,
		`that(` + ss + `).contains_all_of(3)`:                 nil,
		`that(` + ss + `).contains_all_of_in_order(3)`:        nil,
		`that(` + ss + `).contains_all_of(3, [])`:             nil,
		`that(` + ss + `).contains_all_of_in_order(3, [])`:    nil,
		`that(` + ss + `).contains_all_of(3, 5, [])`:          nil,
		`that(` + ss + `).contains_all_of_in_order(3, 5, [])`: nil,
		`that(` + ss + `).contains_all_of([], 3, 5)`:          nil,
		`that(` + ss + `).contains_all_of_in_order([], 3, 5)`: fail(ss,
			`contains all elements in order <([], 3, 5)>`),
		s(`2, 3`):    fail(ss, "contains all of <(2, 3)>. It is missing <2>"),
		s(`2, 3, 6`): fail(ss, "contains all of <(2, 3, 6)>. It is missing <2, 6>"),
	})
}

func TestContainsAllMixedHashableElements(t *testing.T) {
	ss := `(3, [], 5, 8)`
	s := func(x string) string {
		return `that(` + ss + `).contains_all_of(` + x + `)`
	}
	testEach(t, map[string]error{
		`that(` + ss + `).contains_all_of(3, [], 5, 8)`:          nil,
		`that(` + ss + `).contains_all_of_in_order(3, [], 5, 8)`: nil,
		`that(` + ss + `).contains_all_of(5, 3, 8, [])`:          nil,
		`that(` + ss + `).contains_all_of_in_order(5, 3, 8, [])`: fail(ss,
			`contains all elements in order <(5, 3, 8, [])>`),
		s(`3, [], 8, 5, 9`):  fail(ss, "contains all of <(3, [], 8, 5, 9)>. It is missing <9>"),
		s(`3, [], 8, 5, {}`): fail(ss, "contains all of <(3, [], 8, 5, {})>. It is missing <{}>"),
		s(`8, 3, [], 9, 5`):  fail(ss, "contains all of <(8, 3, [], 9, 5)>. It is missing <9>"),
	})
}

func TestContainsAnyIn(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_any_in(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`(3,)`):   nil,
		s(`(7, 3)`): nil,
		s(`()`):     fail(ss, "contains any element in <()>"),
		s(`(2, 6)`): fail(ss, "contains any element in <(2, 6)>"),
	})
}

func TestContainsAnyOf(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_any_of(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):    nil,
		s(`7, 3`): nil,
		s(``):     fail(ss, "contains any of <()>"),
		s(`2, 6`): fail(ss, "contains any of <(2, 6)>"),
	})
}

func TestContainsNoneIn(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_none_in(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`()`):     nil,
		s(`(2,)`):   nil,
		s(`(2, 6)`): nil,
		s(`(5,)`):   fail(ss, "contains no elements in <(5,)>. It contains <5>"),
		s(`(2, 5)`): fail(ss, "contains no elements in <(2, 5)>. It contains <5>"),
	})
}

func TestContainsNoneOf(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `that(` + ss + `).contains_none_of(` + x + `)`
	}
	testEach(t, map[string]error{
		s(``):     nil,
		s(`2`):    nil,
		s(`2, 6`): nil,
		s(`5`):    fail(ss, "contains none of <(5,)>. It contains <5>"),
		s(`2, 5`): fail(ss, "contains none of <(2, 5)>. It contains <5>"),
	})
}

func TestIsOrdered(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_ordered()`
	}
	testEach(t, map[string]error{
		s(`()`):        nil,
		s(`(3,)`):      nil,
		s(`(3, 5, 8)`): nil,
		s(`(3, 5, 5)`): nil,
		s(`(5, 4)`):    newTruthAssertion(`Not true that <(5, 4)> is ordered <(5, 4)>.`),
		s(`(3, 5, 4)`): newTruthAssertion(`Not true that <(3, 5, 4)> is ordered <(5, 4)>.`),
	})
}

func TestIsOrderedAccordingTo(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_ordered_according_to(someCmp)`
	}
	testEach(t, map[string]error{
		s(`()`):        nil,
		s(`(3,)`):      nil,
		s(`(8, 5, 3)`): nil,
		s(`(5, 5, 3)`): nil,
		s(`(4, 5)`):    newTruthAssertion(`Not true that <(4, 5)> is ordered <(4, 5)>.`),
		s(`(3, 5, 4)`): newTruthAssertion(`Not true that <(3, 5, 4)> is ordered <(3, 5)>.`),
	})
}

func TestIsStrictlyOrdered(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_strictly_ordered()`
	}
	testEach(t, map[string]error{
		s(`()`):        nil,
		s(`(3,)`):      nil,
		s(`(3, 5, 8)`): nil,
		s(`(5, 4)`):    newTruthAssertion(`Not true that <(5, 4)> is strictly ordered <(5, 4)>.`),
		s(`(3, 5, 5)`): newTruthAssertion(`Not true that <(3, 5, 5)> is strictly ordered <(5, 5)>.`),
	})
}

func TestIsStrictlyOrderedAccordingTo(t *testing.T) {
	s := func(t string) string {
		return `that(` + t + `).is_strictly_ordered_according_to(someCmp)`
	}
	testEach(t, map[string]error{
		s(`()`):        nil,
		s(`(3,)`):      nil,
		s(`(8, 5, 3)`): nil,
		s(`(4, 5)`):    newTruthAssertion(`Not true that <(4, 5)> is strictly ordered <(4, 5)>.`),
		s(`(5, 5, 3)`): newTruthAssertion(`Not true that <(5, 5, 3)> is strictly ordered <(5, 5)>.`),
	})
}

func TestContainsKey(t *testing.T) {
	ss := `{2: "two", None: "None"}`
	s := func(x string) string {
		return `that(` + ss + `).contains_key(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`2`):     nil,
		s(`None`):  nil,
		s(`3`):     fail(ss, `contains key <3>`),
		s(`"two"`): fail(ss, `contains key <"two">`),
	})
}

func TestDoesNotContainKey(t *testing.T) {
	ss := `{2: "two", None: "None"}`
	s := func(x string) string {
		return `that(` + ss + `).does_not_contain_key(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):     nil,
		s(`"two"`): nil,
		s(`2`):     fail(ss, `does not contain key <2>`),
		s(`None`):  fail(ss, `does not contain key <None>`),
	})
}

func TestContainsItem(t *testing.T) {
	ss := `{2: "two", 4: "four", "too": "two"}`
	s := func(x string) string {
		return `that(` + ss + `).contains_item(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`2, "two"`):     nil,
		s(`4, "four"`):    nil,
		s(`"too", "two"`): nil,
		s(`2, "to"`):      fail(ss, `contains item <(2, "to")>. However, it has a mapping from <2> to <"two">`),
		s(`7, "two"`): fail(ss, `contains item <(7, "two")>.`+
			` However, the following keys are mapped to <"two">: [2, "too"]`),
		s(`7, "seven"`): fail(ss, `contains item <(7, "seven")>`),
	})
}

func TestDoesNotContainItem(t *testing.T) {
	ss := `{2: "two", 4: "four", "too": "two"}`
	s := func(x string) string {
		return `that(` + ss + `).does_not_contain_item(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`2, "to"`):    nil,
		s(`7, "two"`):   nil,
		s(`7, "seven"`): nil,
		s(`2, "two"`):   fail(ss, `does not contain item <(2, "two")>`),
		s(`4, "four"`):  fail(ss, `does not contain item <(4, "four")>`),
	})
}

func TestHasLength(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).has_length(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`): nil,
		s(`4`): fail(ss, `has a length of 4. It is 3`),
		s(`2`): fail(ss, `has a length of 2. It is 3`),
	})
}

func TestStartsWith(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).starts_with(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`""`):   nil,
		s(`"a"`):  nil,
		s(`"ab"`): nil,
		s(abc):    nil,
		s(`"b"`):  fail(ss, `starts with <"b">`),
	})
}

func TestEndsWith(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).ends_with(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`""`):   nil,
		s(`"c"`):  nil,
		s(`"bc"`): nil,
		s(abc):    nil,
		s(`"b"`):  fail(ss, `ends with <"b">`),
	})
}

func TestMatches(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).matches(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"a"`):     nil,
		s(`r".b"`):   nil, // TODO: call re.compile if re module loaded
		s(`r"[Aa]"`): nil, // TODO: use re.I flag if re module loaded
		s(`"d"`):     fail(ss, `matches <"d">`),
		s(`"b"`):     fail(ss, `matches <"b">`),
	})
}

func TestDoesNotMatch(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).does_not_match(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"b"`): nil,
		s(`"d"`): nil,
		s(`"a"`): fail(ss, `fails to match <"a">`),
		s(`r".b"`): fail(ss,
			// TODO: call re.compile if re module loaded
			`fails to match <".b">`),
		s(`r"[Aa]"`): fail(ss,
			// TODO: use re.I flag if re module loaded
			`fails to match <"[Aa]">`),
	})
}

func TestContainsMatch(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).contains_match(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"a"`):     nil,
		s(`r".b"`):   nil, // TODO: call re.compile if re module loaded
		s(`r"[Aa]"`): nil, // TODO: use re.I flag if re module loaded
		s(`"b"`):     nil,
		s(`"d"`):     fail(ss, `should have contained a match for <"d">`),
	})
}

func TestDoesNotContainMatch(t *testing.T) {
	ss := abc
	s := func(x string) string {
		return `that(` + ss + `).does_not_contain_match(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`"d"`): nil,
		s(`"a"`): fail(ss, `should not have contained a match for <"a">`),
		s(`"b"`): fail(ss, `should not have contained a match for <"b">`),
		s(`r".b"`): fail(ss,
			// TODO: call re.compile if re module loaded
			`should not have contained a match for <".b">`),
		s(`r"[Aa]"`): fail(ss,
			// TODO: use re.I flag if re module loaded
			`should not have contained a match for <"[Aa]">`),
	})
}
