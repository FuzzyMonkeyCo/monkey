package starlarktruth

import "testing"

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
