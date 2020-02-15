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
		`AssertThat("abc").containsExactly("abc")`: fail(abc,
			`contains exactly <("abc",)>. It is missing <"abc"> and has unexpected items <"a", "b", "c">`),

		`AssertThat("abc").containsExactly("a", "b", "c")`:           nil,
		`AssertThat("abc").containsExactlyInOrder("a", "b", "c")`:    nil,
		`AssertThat("abc").containsExactly("c", "b", "a")`:           nil,
		`AssertThat("abc").containsExactlyNotInOrder("c", "b", "a")`: nil,

		`AssertThat("abc").containsExactly("a", "bc")`: fail(abc,
			`contains exactly <("a", "bc")>. It is missing <"bc"> and has unexpected items <"b", "c">`),

		`AssertThat(` + tuple + `).containsExactly` + tuple + ``:        nil,
		`AssertThat(` + tuple + `).containsExactly` + elput + ``:        nil,
		`AssertThat(` + tuple + `).containsExactlyInOrder` + tuple + ``: nil,
		// FIXME: impl. inorder
		// `AssertThat(` + tuple + `).containsExactlyInOrder` + elput + ``: fail(tuple,
		// 	`not in order`),

		`AssertThat(` + full + `).containsExactly("a", "` + u1 + `")`: fail(fullTuple,
			`contains exactly <("a", "`+u1+`")>. It has unexpected items <"b", "c", "`+u2+`">`),

		`AssertThat(` + full + `).containsExactly("a` + u1 + `")`: fail(fullTuple,
			`contains exactly <("a`+u1+`",)>. It is missing <"a`+u1+`"> and has unexpected items <"a", "b", "c", "`+u1+`", "`+u2+`">`),
	})
}
