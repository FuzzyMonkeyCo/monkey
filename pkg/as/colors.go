package as

import "github.com/fatih/color"

var (
	// ColorERR colors for errors and fatal messages
	ColorERR = color.New(color.FgRed)
	// ColorWRN colors for warnings and special messages
	ColorWRN = color.New(color.FgYellow)
	// ColorNFO colors for headlines and topical messages
	ColorNFO = color.New(color.Bold)
	// ColorOK colors for valid and successful messages
	ColorOK = color.New(color.FgGreen)
)
