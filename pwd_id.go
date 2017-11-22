package main

import (
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const pwdIDtmpRoot = "/tmp/"

var pwdID string

func makePwdID() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("[ERR] ", err)
	}

	h := fnv.New64a()
	h.Write([]byte(cwd))
	id := pwdIDtmpRoot + "." + binName + "_" + fmt.Sprintf("%d", h.Sum64())

	num := findNewIDSlot(id)

	pwdID = id + "_" + num
}

func findNewIDSlot(prefix string) string {
	prefixPattern := prefix + "_"
	pattern := prefixPattern + strings.Repeat("?", 6) + ".*"
	paths, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatal("[ERR] ", err)
	}

	padder := func(n uint64) string { return fmt.Sprintf("%06d", n) }

	prefixLen := len(prefixPattern)
	nums := []string{padder(0)}
	for _, path := range paths {
		nums = append(nums, path[prefixLen:prefixLen+6])
	}
	sort.Strings(nums)

	biggest := nums[len(nums)-1]
	big, err := strconv.ParseUint(biggest, 10, 32)
	if err != nil {
		log.Fatal("[ERR] ", err)
	}

	return padder(big + 1)
}
