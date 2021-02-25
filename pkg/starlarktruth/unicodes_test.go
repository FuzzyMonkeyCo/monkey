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
		`that("abc").contains_exactly("abc")`: fail(abc,
			`contains exactly <("abc",)>. It is missing <"abc"> and has unexpected items <"a", "b", "c">`),

		`that("abc").contains_exactly("a", "b", "c")`:          nil,
		`that("abc").contains_exactly_in_order("a", "b", "c")`: nil,
		`that("abc").contains_exactly("c", "b", "a")`:          nil,
		`that("abc").contains_exactly_in_order("c", "b", "a")`: fail(abc,
			`contains exactly these elements in order <("c", "b", "a")>`),

		`that("abc").contains_exactly("a", "bc")`: fail(abc,
			`contains exactly <("a", "bc")>. It is missing <"bc"> and has unexpected items <"b", "c">`),

		`that(` + tuple + `).contains_exactly` + tuple + ``:          nil,
		`that(` + tuple + `).contains_exactly` + elput + ``:          nil,
		`that(` + tuple + `).contains_exactly_in_order` + tuple + ``: nil,
		`that(` + tuple + `).contains_exactly_in_order` + elput + ``: fail(tuple,
			`contains exactly these elements in order <`+elput+`>`),

		`that(` + full + `).contains_exactly("a", "` + u1 + `")`: fail(fullTuple,
			`contains exactly <("a", "`+u1+`")>. It has unexpected items <"b", "c", "`+u2+`">`),

		`that(` + full + `).contains_exactly("a` + u1 + `")`: fail(fullTuple,
			`contains exactly <("a`+u1+`",)>. It is missing <"a`+u1+`"> and has unexpected items <"a", "b", "c", "`+u1+`", "`+u2+`">`),
	})
}
