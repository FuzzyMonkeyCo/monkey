package cwid

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func filesContain(t *testing.T, pattern string) {
	t.Helper()

	require.Contains(t, LogFile(), pattern)
	require.Contains(t, Prefixed(), pattern)
}

func TestPwdID(t *testing.T) {
	if os.Getenv("TESTPWDID") != "1" {
		t.Skipf("Run these tests under docker buildx")
	}

	err := copyFile("../../fuzzymonkey.star", "fuzzymonkey.star")
	require.NoError(t, err)
	defer os.Remove("fuzzymonkey.star")
	err = copyFile("../../README.md", "README.md")
	require.NoError(t, err)
	defer os.Remove("README.md")

	const name, starfile, offset = "monkeh", "fuzzymonkey.star", 0

	err = MakePwdID(name, starfile, offset)
	require.NoError(t, err)
	filesContain(t, ".monkeh_")
	filesContain(t, "_12404825836092798244_")
	filesContain(t, "_00000000000000000001")

	// PwdID changes with offset

	func(filename string) {

		newfilename := strings.Replace(filename, "_00000000000000000001", "_00000000000000000002", -1)
		err := os.WriteFile(newfilename, nil, 0644)
		require.NoError(t, err)
		defer os.Remove(newfilename)

		require.NotEqual(t, 1, offset)
		err = MakePwdID(name, starfile, 1)
		require.NoError(t, err)
		filesContain(t, ".monkeh_")
		filesContain(t, "_12404825836092798244_")
		filesContain(t, "_00000000000000000002")

	}(LogFile())

	// PwdID changes with starfile

	require.NotEqual(t, "README.md", starfile)
	err = MakePwdID(name, "README.md", offset)
	require.NoError(t, err)
	filesContain(t, ".monkeh_")
	filesContain(t, "_2078280350767222314_")
	filesContain(t, "_00000000000000000001")

	// No symlinks allowed
	func() {

		err := os.Symlink(starfile, "fm.star")
		require.NoError(t, err)
		defer os.Remove("fm.star")

		require.NotEqual(t, "fm.star", starfile)
		err = MakePwdID(name, "./fm.star", offset)
		require.EqualError(t, err, `is a symlink: "./fm.star"`)

	}()

	// No lurking outside the nest

	require.NotEqual(t, "../../main.go", starfile)
	err = MakePwdID(name, "../../main.go", offset)
	require.EqualError(t, err, `must be in current directory: "../../main.go"`)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
