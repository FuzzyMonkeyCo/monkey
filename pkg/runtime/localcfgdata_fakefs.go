//go:build fakefs
// +build fakefs

package runtime

import (
	"io/ioutil"
)

var localCfgData []byte

func localcfgdata() ([]byte, error) {
	if localCfgData != nil {
		return localCfgData, nil
	}
	return ioutil.ReadFile(localCfg)
}
