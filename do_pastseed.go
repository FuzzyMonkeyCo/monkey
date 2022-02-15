package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/FuzzyMonkeyCo/monkey/pkg/code"
	"github.com/FuzzyMonkeyCo/monkey/pkg/cwid"
	rt "github.com/FuzzyMonkeyCo/monkey/pkg/runtime"
)

var rePastseed = regexp.MustCompile(rt.PastSeedMagic + `=([^\s]+)`)

// Looks in the logs for the youngest seed that triggered a bug
// Only ever prints best seed on a newline character
// so it can be used as --seed=$(monkey pastseed)
func doPastseed(starfile string) int {
	for offset := uint64(1); true; offset++ {
		var seed string
		ret := -1
		func() {
			if err := cwid.MakePwdID(binName, starfile, offset); err != nil {
				// Fails silently
				ret = code.Failed
				return
			}

			fn := cwid.LogFile()
			f, err := os.Open(fn)
			if err != nil {
				ret = code.Failed
				return
			}
			defer f.Close()

			s := bufio.NewScanner(f)
			for s.Scan() {
				matches := rePastseed.FindStringSubmatch(s.Text())
				if len(matches) != 0 && matches[1] != "" {
					seed = matches[1]
				}
			}
		}()
		if ret != -1 {
			return ret
		}
		if seed != "" {
			fmt.Println(seed)
			return code.OK
		}
	}
	return code.Failed
}
