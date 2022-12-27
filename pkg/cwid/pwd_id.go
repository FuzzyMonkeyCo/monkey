package cwid

import (
	"fmt"
	"hash/fnv"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const pwdIDDigits = 20

var pwdID string

// EnvFile points to a usable regular file after a call to MakePwdID()
func EnvFile() string { return pwdID + ".env" }

// LogFile points to a usable regular file after a call to MakePwdID()
func LogFile() string { return pwdID + ".log" }

// ScriptFile points to a usable regular file after a call to MakePwdID()
func ScriptFile() string { return pwdID + ".script" }

// MakePwdID looks for a usable temporary path for var run logfiles
func MakePwdID(name, starfile string, offset uint64) (err error) {
	var fi os.FileInfo
	if fi, err = os.Lstat(starfile); err != nil {
		return
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		err = fmt.Errorf("is a symlink: %q", starfile)
		return
	}

	if path.Clean(starfile) != path.Base(starfile) {
		err = fmt.Errorf("must be in current directory: %q", starfile)
		return
	}

	var cwd string
	if cwd, err = os.Getwd(); err != nil {
		return
	}
	if cwd, err = filepath.EvalSymlinks(cwd); err != nil {
		return
	}

	h := fnv.New64a()
	if _, err = h.Write([]byte(cwd)); err != nil {
		return
	}
	if _, err = h.Write([]byte("/")); err != nil {
		return
	}
	if _, err = h.Write([]byte(path.Base(starfile))); err != nil {
		return
	}
	id := fmt.Sprintf("%d", h.Sum64())

	tmp := os.TempDir()
	if err = os.MkdirAll(tmp, 0700); err != nil {
		return
	}

	prefix := path.Join(tmp, "."+name+"_"+id)

	var slot string
	if slot, err = findIDSlot(prefix, offset); err != nil {
		return
	}

	pwdID = prefix + "_" + slot
	return
}

func findIDSlot(prefix string, offset uint64) (slot string, err error) {
	prefixPattern := prefix + "_"
	pattern := prefixPattern + strings.Repeat("?", pwdIDDigits) + ".*"
	var paths []string
	if paths, err = filepath.Glob(pattern); err != nil {
		return
	}

	padder := func(n uint64) string {
		return fmt.Sprintf("%0"+strconv.Itoa(pwdIDDigits)+"d", n)
	}

	prefixLen := len(prefixPattern)
	nums := []string{padder(0)}
	for _, path := range paths {
		nums = append(nums, path[prefixLen:prefixLen+pwdIDDigits])
	}
	sort.Strings(nums)

	biggest := nums[len(nums)-1]
	var big uint64
	if big, err = strconv.ParseUint(biggest, 10, 32); err != nil {
		return
	}

	slot = padder(big + 1 - offset)
	return
}
