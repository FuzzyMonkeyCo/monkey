//go:build !fakefs
// +build !fakefs

package runtime

import "io/ioutil"

func localcfgdata() ([]byte, error) { return ioutil.ReadFile(localCfg) }
