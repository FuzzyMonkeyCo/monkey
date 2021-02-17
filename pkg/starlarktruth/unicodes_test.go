package starlarktruth

import "testing"

func TestContainsExactlyHandlesStringsAsCodepoints(t *testing.T) {
	const (
		// multiple bytes codepoint
		u1 = `Й`
		// more multiple bytes codepoint
		u2 = `😿`
		// concats
		full      = `"abc` + u1 + u2 + `"`
		fullTuple = `("a", "b", "c", "` + u1 + `", "` + u2 + `")`
		tuple     = `("a", "` + u1 + `", "c")`
		elput     = `("c", "` + u1 + `", "a")`
		abc       = `("a", "b", "c")`
	)
	testEach(t, map[string]error{
		`assert.that("abc").contains_exactly("abc")`: fail(abc,
			`contains exactly <("abc",)>. It is missing <"abc"> and has unexpected items <"a", "b", "c">`),

		`assert.that("abc").contains_exactly("a", "b", "c")`:          nil,
		`assert.that("abc").contains_exactly_in_order("a", "b", "c")`: nil,
		`assert.that("abc").contains_exactly("c", "b", "a")`:          nil,
		`assert.that("abc").contains_exactly_in_order("c", "b", "a")`: fail(abc,
			`contains exactly these elements in order <("c", "b", "a")>`),

		`assert.that("abc").contains_exactly("a", "bc")`: fail(abc,
			`contains exactly <("a", "bc")>. It is missing <"bc"> and has unexpected items <"b", "c">`),

		`assert.that(` + tuple + `).contains_exactly` + tuple + ``:          nil,
		`assert.that(` + tuple + `).contains_exactly` + elput + ``:          nil,
		`assert.that(` + tuple + `).contains_exactly_in_order` + tuple + ``: nil,
		`assert.that(` + tuple + `).contains_exactly_in_order` + elput + ``: fail(tuple,
			`contains exactly these elements in order <`+elput+`>`),

		`assert.that(` + full + `).contains_exactly("a", "` + u1 + `")`: fail(fullTuple,
			`contains exactly <("a", "`+u1+`")>. It has unexpected items <"b", "c", "`+u2+`">`),

		`assert.that(` + full + `).contains_exactly("a` + u1 + `")`: fail(fullTuple,
			`contains exactly <("a`+u1+`",)>. It is missing <"a`+u1+`"> and has unexpected items <"a", "b", "c", "`+u1+`", "`+u2+`">`),
	})
}
