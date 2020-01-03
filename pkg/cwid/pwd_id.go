package cwid

import (
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const pwdIDDigits = 20

var pwdID string

func EnvFile() string {
	return pwdID + ".env"
}

func LogFile() string {
	return pwdID + ".log"
}

func MakePwdID(name string, offset uint64) (err error) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	realCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	tmp := os.TempDir()
	if err = os.MkdirAll(tmp, 0700); err != nil {
		log.Println("[ERR]", err)
		return
	}
	h := fnv.New64a()
	if _, err = h.Write([]byte(realCwd)); err != nil {
		log.Println("[ERR]", err)
		return
	}
	id := fmt.Sprintf("%d", h.Sum64())
	prefix := path.Join(tmp, "."+name+"_"+id)

	slot, err := findIDSlot(prefix, offset)
	if err != nil {
		return
	}

	pwdID = prefix + "_" + slot
	return
}

func findIDSlot(prefix string, offset uint64) (slot string, err error) {
	prefixPattern := prefix + "_"
	pattern := prefixPattern + strings.Repeat("?", pwdIDDigits) + ".*"
	paths, err := filepath.Glob(pattern)
	if err != nil {
		log.Println("[ERR]", err)
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
	big, err := strconv.ParseUint(biggest, 10, 32)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	slot = padder(big + 1 - offset)
	return
}
