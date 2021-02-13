package starlarktruth

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

const abc = `"abc"` // Please linter

func helper(t *testing.T, program string) (starlark.StringDict, error) {
	// Enabled so they can be tested
	resolve.AllowFloat = true
	resolve.AllowSet = true
	resolve.AllowLambda = true

	predeclared := starlark.StringDict{}
	NewModule(predeclared)
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
	return starlark.ExecFile(thread, t.Name()+".star", script, predeclared)
}

func testEach(t *testing.T, m map[string]error) {
	for code, expectedErr := range m {
		t.Run(code, func(t *testing.T) {
			globals, err := helper(t, code)
			delete(globals, "dfltCmp")
			delete(globals, "someCmp")
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

func TestTrue(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(True).isTrue()`:  nil,
		`AssertThat(True).isFalse()`: fail("True", "is False"),
	})
}

func TestFalse(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(False).isFalse()`: nil,
		`AssertThat(False).isTrue()`:  fail("False", "is True"),
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
		m[`AssertThat(`+v+`).isTruthy()`] = nil
		m[`AssertThat(`+v+`).isFalsy()`] = fail(v, "is falsy")
		m[`AssertThat(`+v+`).isFalse()`] = fail(v, "is False")
		if v != `True` {
			m[`AssertThat(`+v+`).isTrue()`] = fail(v, "is True",
				" However, it is truthy. Did you mean to call .isTruthy() instead?")
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
		m[`AssertThat(`+v+`).isFalsy()`] = nil
		m[`AssertThat(`+v+`).isTruthy()`] = fail(vv, "is truthy")
		m[`AssertThat(`+v+`).isTrue()`] = fail(vv, "is True")
		if v != `False` {
			m[`AssertThat(`+v+`).isFalse()`] = fail(vv, "is False",
				" However, it is falsy. Did you mean to call .isFalsy() instead?")
		}
	}
	testEach(t, m)
}

func TestIsAtLeast(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isAtLeast(3)`: nil,
		`AssertThat(5).isAtLeast(5)`: nil,
		`AssertThat(5).isAtLeast(8)`: fail("5", "is at least <8>"),
	})
}

func TestIsAtMost(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isAtMost(5)`: nil,
		`AssertThat(5).isAtMost(8)`: nil,
		`AssertThat(5).isAtMost(3)`: fail("5", "is at most <3>"),
	})
}

func TestIsGreaterThan(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isGreaterThan(3)`: nil,
		`AssertThat(5).isGreaterThan(5)`: fail("5", "is greater than <5>"),
		`AssertThat(5).isGreaterThan(8)`: fail("5", "is greater than <8>"),
	})
}

func TestIsLessThan(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isLessThan(8)`: nil,
		`AssertThat(5).isLessThan(5)`: fail("5", "is less than <5>"),
		`AssertThat(5).isLessThan(3)`: fail("5", "is less than <3>"),
	})
}

func TestCannotCompareToNone(t *testing.T) {
	p := "It is illegal to compare using ."
	testEach(t, map[string]error{
		`AssertThat(5).isAtLeast(None)`:     newInvalidAssertion(p + "isAtLeast(None)"),
		`AssertThat(5).isAtMost(None)`:      newInvalidAssertion(p + "isAtMost(None)"),
		`AssertThat(5).isGreaterThan(None)`: newInvalidAssertion(p + "isGreaterThan(None)"),
		`AssertThat(5).isLessThan(None)`:    newInvalidAssertion(p + "isLessThan(None)"),
	})
}

func TestIsEqualTo(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isEqualTo(5)`: nil,
		`AssertThat(5).isEqualTo(3)`: fail("5", "is equal to <3>"),
		`AssertThat({1:2,3:4}).isEqualTo([1,2,3,4])`: fail(`{1: 2, 3: 4}`,
			"is equal to <[1, 2, 3, 4]>"),
	})
}

func TestIsEqualToFailsOnFloatsAsWellAsWithFormattedRepresentations(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(0.3).isEqualTo(0.1+0.2)`: fail("0.3", "is equal to <0.30000000000000004>"),
		`AssertThat(0.1+0.2).isEqualTo(0.3)`: fail("0.30000000000000004", "is equal to <0.3>"),
	})
}

func TestIsNotEqualTo(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(5).isNotEqualTo(3)`: nil,
		`AssertThat(5).isNotEqualTo(5)`: fail("5", "is not equal to <5>"),
	})
}

func TestSequenceIsEqualToUsesContainsExactlyElementsInPlusInOrder(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat((3,5,[])).isEqualTo((3, 5, []))`: nil,
		`AssertThat((3,5,[])).isEqualTo(([],3,5))`: fail("(3, 5, [])",
			"contains exactly these elements in order <([], 3, 5)>"),
		`AssertThat((3,5,[])).isEqualTo((3,5,[],9))`: fail("(3, 5, [])",
			"contains exactly <(3, 5, [], 9)>. It is missing <9>"),
		`AssertThat((3,5,[])).isEqualTo((9,3,5,[],10))`: fail("(3, 5, [])",
			"contains exactly <(9, 3, 5, [], 10)>. It is missing <9, 10>"),
		`AssertThat((3,5,[])).isEqualTo((3,5))`: fail("(3, 5, [])",
			"contains exactly <(3, 5)>. It has unexpected items <[]>"),
		`AssertThat((3,5,[])).isEqualTo(([],3))`: fail("(3, 5, [])",
			"contains exactly <([], 3)>. It has unexpected items <5>"),
		`AssertThat((3,5,[])).isEqualTo((3,))`: fail("(3, 5, [])",
			"contains exactly <(3,)>. It has unexpected items <5, []>"),
		`AssertThat((3,5,[])).isEqualTo((4,4,3,[],5))`: fail("(3, 5, [])",
			"contains exactly <(4, 4, 3, [], 5)>. It is missing <4 [2 copies]>"),
		`AssertThat((3,5,[])).isEqualTo((4,4))`: fail("(3, 5, [])",
			"contains exactly <(4, 4)>. It is missing <4 [2 copies]> and has unexpected items <3, 5, []>"),
		`AssertThat((3,5,[])).isEqualTo((3,5,9))`: fail("(3, 5, [])",
			"contains exactly <(3, 5, 9)>. It is missing <9> and has unexpected items <[]>"),
		`AssertThat((3,5,[])).isEqualTo(())`: fail("(3, 5, [])", "is empty"),
	})
}

func TestSetIsEqualToUsesContainsExactlyElementsIn(t *testing.T) {
	s := `AssertThat(set([3, 5, 8]))`
	testEach(t, map[string]error{
		s + `.isEqualTo(set([3, 5, 8]))`: nil,
		s + `.isEqualTo(set([8, 3, 5]))`: nil,
		s + `.isEqualTo(set([3, 5, 8, 9]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5, 8, 9])>. It is missing <9>"),
		s + `.isEqualTo(set([9, 3, 5, 8, 10]))`: fail("set([3, 5, 8])",
			"contains exactly <set([9, 3, 5, 8, 10])>. It is missing <9, 10>"),
		s + `.isEqualTo(set([3, 5]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5])>. It has unexpected items <8>"),
		s + `.isEqualTo(set([8, 3]))`: fail("set([3, 5, 8])",
			"contains exactly <set([8, 3])>. It has unexpected items <5>"),
		s + `.isEqualTo(set([3]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3])>. It has unexpected items <5, 8>"),
		s + `.isEqualTo(set([4]))`: fail("set([3, 5, 8])",
			"contains exactly <set([4])>. It is missing <4> and has unexpected items <3, 5, 8>"),
		s + `.isEqualTo(set([3, 5, 9]))`: fail("set([3, 5, 8])",
			"contains exactly <set([3, 5, 9])>. It is missing <9> and has unexpected items <8>"),
		s + `.isEqualTo(set([]))`: fail("set([3, 5, 8])", "is empty"),
	})
}

func TestSequenceIsEqualToComparedWithNonIterables(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat((3, 5, [])).isEqualTo(3)`: fail("(3, 5, [])", "is equal to <3>"),
	})
}

func TestSetIsEqualToComparedWithNonIterables(t *testing.T) {
	testEach(t, map[string]error{
		`AssertThat(set([3, 5, 8])).isEqualTo(3)`: fail("set([3, 5, 8])", "is equal to <3>"),
	})
}

func TestOrderedDictIsEqualToUsesContainsExactlyItemsInPlusInOrder(t *testing.T) {
	d1 := `((2, "two"), (4, "four"))`
	d2 := `((2, "two"), (4, "four"))`
	d3 := `((4, "four"), (2, "two"))`
	d4 := `((2, "two"), (4, "for"))`
	d5 := `((2, "two"), (4, "four"), (5, "five"))`
	s := `AssertThat(` + d1 + `).isEqualTo(`
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
	s := `AssertThat(` + d + `).isEqualTo(`
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
		`AssertThat({2:"two",4:"four"}).isEqualTo(3)`: fail(
			`{2: "two", 4: "four"}`,
			`is equal to <3>`,
		),
	})
}

func TestNamedMultilineString(t *testing.T) {
	s := `AssertThat("line1\nline2").named("some-name")`
	testEach(t, map[string]error{
		s + `.isEqualTo("line1\nline2")`: nil,
		s + `.isEqualTo("")`: newTruthAssertion(
			`Not true that actual some-name is equal to <"">.`),
		s + `.isEqualTo("line1\nline2\n")`: newTruthAssertion(
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
		`AssertThat("line1\nline2\nline3\nline4\nline5\n") \
         .isEqualTo("line1\nline2\nline4\nline6\n")`: newTruthAssertion(
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
		return `AssertThat(` + ss + `).containsExactly(` + x + `)`
	}
	testEach(t, map[string]error{
		`AssertThat(` + ss + `).containsExactlyInOrder(3, 5, [])`: nil,
		`AssertThat(` + ss + `).containsExactly(3, 5, [])`:        nil,
		`AssertThat(` + ss + `).containsExactly([], 3, 5)`:        nil,
		`AssertThat(` + ss + `).containsExactlyInOrder([], 3, 5)`: fail(ss,
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
	s := `.containsExactly("abc")`
	testEach(t, map[string]error{
		`AssertThat(())` + s:      fail(`()`, `contains exactly <("abc",)>. It is missing <"abc">`),
		`AssertThat([])` + s:      fail(`[]`, `contains exactly <("abc",)>. It is missing <"abc">`),
		`AssertThat({})` + s:      errMustBeEqualNumberOfKVPairs(1),
		`AssertThat("")` + s:      fail(`""`, `contains exactly <("abc",)>. It is missing <"abc">`),
		`AssertThat(set([]))` + s: fail(`set([])`, `contains exactly <("abc",)>. It is missing <"abc">`),
	})
}

func TestContainsExactlyEmptyContainer(t *testing.T) {
	s := func(x string) string {
		return `AssertThat(` + x + `).containsExactly(3)`
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
		return `AssertThat(` + x + `).containsExactly()`
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
		return `AssertThat(` + ss + `).containsExactlyElementsIn(` + x + `)`
	}
	testEach(t, map[string]error{
		`AssertThat(` + ss + `).containsExactlyElementsInOrderIn((3, 5, []))`: nil,
		`AssertThat(` + ss + `).containsExactlyElementsIn(([], 3, 5))`:        nil,
		`AssertThat(` + ss + `).containsExactlyElementsInOrderIn(([], 3, 5))`: fail(ss,
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
		`AssertThat(()).containsExactlyElementsIn(())`: nil,
		`AssertThat(()).containsExactlyElementsIn((3,))`: fail(`()`,
			`contains exactly <(3,)>. It is missing <3>`),
	})
}

func TestContainsExactlyTargetingOrderedDict(t *testing.T) {
	ss := `((2, "two"), (4, "four"))`
	s := `AssertThat(` + ss + `).containsExactly(`
	testEach(t, map[string]error{
		`AssertThat(` + ss + `).containsExactlyInOrder((2, "two"), (4, "four"))`: nil,
		`AssertThat(` + ss + `).containsExactly((2, "two"), (4, "four"))`:        nil,
		`AssertThat(` + ss + `).containsExactly((4, "four"), (2, "two"))`:        nil,
		`AssertThat(` + ss + `).containsExactlyInOrder((4, "four"), (2, "two"))`: fail(ss,
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
		`AssertThat({}).containsExactly("key1", "value1", "key2")`: errMustBeEqualNumberOfKVPairs(3),
	})
}

func TestContainsExactlyItemsIn(t *testing.T) {
	s := func(x string) string {
		return `AssertThat({2: "two", 4: "four"}).containsExactlyItemsIn(` + x + `)`
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
		`AssertThat(None).isNone()`:     nil,
		`AssertThat(None).isNotNone()`:  fail(`None`, `is not None`),
		`AssertThat("abc").isNotNone()`: nil,
		`AssertThat("abc").isNone()`:    fail(abc, `is None`),
	})
}

func TestIsIn(t *testing.T) {
	s := func(x string) string {
		return `AssertThat(3).isIn(` + x + `)`
	}
	testEach(t, map[string]error{
		`AssertThat("a").isIn("abc")`: nil,
		`AssertThat("d").isIn("abc")`: fail(`"d"`, `is equal to any of <"abc">`),
		s(`(3,)`):                     nil,
		s(`(3, 5)`):                   nil,
		s(`(1, 3, 5)`):                nil,
		s(`{3: "three"}`):             nil,
		s(`set([3, 5])`):              nil,
		s(`()`):                       fail(`3`, `is equal to any of <()>`),
		s(`(2,)`):                     fail(`3`, `is equal to any of <(2,)>`),
	})
}

func TestIsNotIn(t *testing.T) {
	s := func(x string) string {
		return `AssertThat(3).isNotIn(` + x + `)`
	}
	testEach(t, map[string]error{
		`AssertThat("a").isNotIn("abc")`: fail(`"a"`, `is not in "abc". It was found at index 0`),
		`AssertThat("d").isNotIn("abc")`: nil,
		s(`(5,)`):                        nil,
		s(`set([5])`):                    nil,
		s(`("3",)`):                      nil,
		s(`(3,)`):                        fail(`3`, `is not in (3,). It was found at index 0`),
		s(`(1, 3)`):                      fail(`3`, `is not in (1, 3). It was found at index 1`),
		s(`set([3])`):                    fail(`3`, `is not in set([3])`),
	})
}

func TestIsAnyOf(t *testing.T) {
	s := func(x string) string {
		return `AssertThat(3).isAnyOf(` + x + `)`
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
		return `AssertThat(3).isNoneOf(` + x + `)`
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
		return `AssertThat("my str").hasAttribute(` + x + `)`
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
		return `AssertThat({1: ()}).doesNotHaveAttribute(` + x + `)`
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
		return `AssertThat(` + t + `).isCallable()`
	}
	testEach(t, map[string]error{
		s(`lambda x: x`):    nil,
		s(`"str".endswith`): nil,
		s(`AssertThat`):     nil,
		s(`None`):           fail(`None`, `is callable`),
		s(abc):              fail(abc, `is callable`),
	})
}

func TestIsNotCallable(t *testing.T) {
	s := func(t string) string {
		return `AssertThat(` + t + `).isNotCallable()`
	}
	testEach(t, map[string]error{
		s(`None`):           nil,
		s(abc):              nil,
		s(`lambda x: x`):    fail(`function lambda`, `is not callable`),
		s(`"str".endswith`): fail(`built-in method endswith of string value`, `is not callable`),
		s(`AssertThat`):     fail(`built-in function AssertThat`, `is not callable`),
	})
}

func TestHasSize(t *testing.T) {
	s := func(x string) string {
		return `AssertThat((2, 5, 8)).hasSize(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`3`):  nil,
		s(`-1`): fail(`(2, 5, 8)`, `has a size of <-1>. It is <3>`),
		s(`2`):  fail(`(2, 5, 8)`, `has a size of <2>. It is <3>`),
	})
}

func TestIsEmpty(t *testing.T) {
	s := func(t string) string {
		return `AssertThat(` + t + `).isEmpty()`
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
		return `AssertThat(` + t + `).isNotEmpty()`
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
		return `AssertThat((2, 5, [])).contains(` + x + `)`
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
		return `AssertThat((2, 5, [])).doesNotContain(` + x + `)`
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
		return `AssertThat(` + t + `).containsNoDuplicates()`
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
		return `AssertThat(` + ss + `).containsAllIn(` + x + `)`
	}
	testEach(t, map[string]error{
		s(`()`): nil,
		`AssertThat(` + ss + `).containsAllIn(())`:                nil,
		`AssertThat(` + ss + `).containsAllInOrderIn(())`:         nil,
		`AssertThat(` + ss + `).containsAllIn((3,))`:              nil,
		`AssertThat(` + ss + `).containsAllInOrderIn((3,))`:       nil,
		`AssertThat(` + ss + `).containsAllIn((3, []))`:           nil,
		`AssertThat(` + ss + `).containsAllInOrderIn((3, []))`:    nil,
		`AssertThat(` + ss + `).containsAllIn((3, 5, []))`:        nil,
		`AssertThat(` + ss + `).containsAllInOrderIn((3, 5, []))`: nil,
		`AssertThat(` + ss + `).containsAllIn(([], 5, 3))`:        nil,
		`AssertThat(` + ss + `).containsAllInOrderIn(([], 5, 3))`: fail(ss,
			`contains all elements in order <([], 5, 3)>`),
		s(`(2, 3)`):    fail(ss, "contains all elements in <(2, 3)>. It is missing <2>"),
		s(`(2, 3, 6)`): fail(ss, "contains all elements in <(2, 3, 6)>. It is missing <2, 6>"),
	})
}

func TestContainsAllOf(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `AssertThat(` + ss + `).containsAllOf(` + x + `)`
	}
	testEach(t, map[string]error{
		`AssertThat(` + ss + `).containsAllOf()`:                nil,
		`AssertThat(` + ss + `).containsAllOfInOrder()`:         nil,
		`AssertThat(` + ss + `).containsAllOf(3)`:               nil,
		`AssertThat(` + ss + `).containsAllOfInOrder(3)`:        nil,
		`AssertThat(` + ss + `).containsAllOf(3, [])`:           nil,
		`AssertThat(` + ss + `).containsAllOfInOrder(3, [])`:    nil,
		`AssertThat(` + ss + `).containsAllOf(3, 5, [])`:        nil,
		`AssertThat(` + ss + `).containsAllOfInOrder(3, 5, [])`: nil,
		`AssertThat(` + ss + `).containsAllOf([], 3, 5)`:        nil,
		`AssertThat(` + ss + `).containsAllOfInOrder([], 3, 5)`: fail(ss,
			`contains all elements in order <([], 3, 5)>`),
		s(`2, 3`):    fail(ss, "contains all of <(2, 3)>. It is missing <2>"),
		s(`2, 3, 6`): fail(ss, "contains all of <(2, 3, 6)>. It is missing <2, 6>"),
	})
}

func TestContainsAllMixedHashableElements(t *testing.T) {
	ss := `(3, [], 5, 8)`
	s := func(x string) string {
		return `AssertThat(` + ss + `).containsAllOf(` + x + `)`
	}
	testEach(t, map[string]error{
		`AssertThat(` + ss + `).containsAllOf(3, [], 5, 8)`:        nil,
		`AssertThat(` + ss + `).containsAllOfInOrder(3, [], 5, 8)`: nil,
		`AssertThat(` + ss + `).containsAllOf(5, 3, 8, [])`:        nil,
		`AssertThat(` + ss + `).containsAllOfInOrder(5, 3, 8, [])`: fail(ss,
			`contains all elements in order <(5, 3, 8, [])>`),
		s(`3, [], 8, 5, 9`):  fail(ss, "contains all of <(3, [], 8, 5, 9)>. It is missing <9>"),
		s(`3, [], 8, 5, {}`): fail(ss, "contains all of <(3, [], 8, 5, {})>. It is missing <{}>"),
		s(`8, 3, [], 9, 5`):  fail(ss, "contains all of <(8, 3, [], 9, 5)>. It is missing <9>"),
	})
}

func TestContainsAnyIn(t *testing.T) {
	ss := `(3, 5, [])`
	s := func(x string) string {
		return `AssertThat(` + ss + `).containsAnyIn(` + x + `)`
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
		return `AssertThat(` + ss + `).containsAnyOf(` + x + `)`
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
		return `AssertThat(` + ss + `).containsNoneIn(` + x + `)`
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
		return `AssertThat(` + ss + `).containsNoneOf(` + x + `)`
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
		return `AssertThat(` + t + `).isOrdered()`
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
		return `AssertThat(` + t + `).isOrderedAccordingTo(someCmp)`
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
		return `AssertThat(` + t + `).isStrictlyOrdered()`
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
		return `AssertThat(` + t + `).isStrictlyOrderedAccordingTo(someCmp)`
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
		return `AssertThat(` + ss + `).containsKey(` + x + `)`
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
		return `AssertThat(` + ss + `).doesNotContainKey(` + x + `)`
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
		return `AssertThat(` + ss + `).containsItem(` + x + `)`
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
		return `AssertThat(` + ss + `).doesNotContainItem(` + x + `)`
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
		return `AssertThat(` + ss + `).hasLength(` + x + `)`
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
		return `AssertThat(` + ss + `).startsWith(` + x + `)`
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
		return `AssertThat(` + ss + `).endsWith(` + x + `)`
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
		return `AssertThat(` + ss + `).matches(` + x + `)`
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
		return `AssertThat(` + ss + `).doesNotMatch(` + x + `)`
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
		return `AssertThat(` + ss + `).containsMatch(` + x + `)`
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
		return `AssertThat(` + ss + `).doesNotContainMatch(` + x + `)`
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
