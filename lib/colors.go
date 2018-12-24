package lib

import (
	"github.com/fatih/color"
)

var (
	ColorERR *color.Color
	ColorWRN *color.Color
	ColorNFO *color.Color
)

func init() {
	ColorERR = color.New(color.FgRed)
	ColorWRN = color.New(color.FgYellow)
	ColorNFO = color.New(color.Bold)
}
