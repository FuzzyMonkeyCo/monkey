package runtime

import (
	"testing"

	"github.com/bazelbuild/buildtools/warn"
	"github.com/stretchr/testify/require"
)

// https://github.com/bazelbuild/buildtools/blob/f1ead6bc540dfa6ed95dfb4fe5f0c0bfc9243370/WARNINGS.md#buildifier-warnings

func TestFmtWarnings(t *testing.T) {
	require.Subset(t, warn.AllWarnings, warn.DefaultWarnings)
	require.Subset(t, warn.AllWarnings, fmtWarningsList)
	require.IsIncreasing(t, fmtWarningsList)

	deny := []string{
		"attr-cfg",
		"attr-license",
		"attr-non-empty",
		"attr-output-default",
		"attr-single-file",
		"ctx-actions",
		"ctx-args",
		"filetype",
		"git-repository",
		"http-archive",
		"load",
		"load-on-top",
		"module-docstring",
		"native-android",
		"native-build",
		"native-cc",
		"native-java",
		"native-package",
		"native-proto",
		"native-py",
		"out-of-order-load",
		"output-group",
		"package-name",
		"package-on-top",
		"provider-params",
		"repository-name",
		"rule-impl-return",
		"same-origin-load",
	}
	require.Subset(t, warn.AllWarnings, deny)
	require.IsIncreasing(t, deny)

	for _, w := range warn.AllWarnings {
		for _, d := range deny {
			if d == w {
				require.NotContains(t, fmtWarningsList, w)
				break
			}
		}
	}
}
