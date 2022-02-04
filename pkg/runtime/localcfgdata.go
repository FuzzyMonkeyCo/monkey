package runtime

import (
	"os"
)

var localCfgData []byte

func localcfgdata() (data []byte, err error) {
	data = localCfgData
	if localCfgData == nil { // When not mocking
		if data, err = os.ReadFile(localCfg); err != nil {
			return
		}
	}

	data = starTrick(data)
	return
}
