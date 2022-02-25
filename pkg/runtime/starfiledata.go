package runtime

import (
	"os"
)

var starlarkCompareLimit int

var starfileData []byte

func starfiledata(starfile string) (data []byte, err error) {
	data = starfileData
	if starfileData == nil { // When not mocking
		if data, err = os.ReadFile(starfile); err != nil {
			return
		}
	}

	data = starTrick(data)
	return
}
