package runtime

import "os"

var localCfgData []byte

func localcfgdata() ([]byte, error) {
	if localCfgData != nil {
		// When mocking
		return localCfgData, nil
	}

	return os.ReadFile(localCfg)
}
