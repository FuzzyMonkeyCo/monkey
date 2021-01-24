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

func doPastseed() int {
	// Looks in the logs for the youngest seed that triggered a bug
	// Prints nothing on error so it can be used as --seed=$(monkey pastseed)
	for offset := uint64(1); true; offset++ {
		var seed string
		ret := -1
		func() {
			if err := cwid.MakePwdID(binName, offset); err != nil {
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
