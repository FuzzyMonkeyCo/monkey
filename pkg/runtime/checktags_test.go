//go:build !fakefs
// +build !fakefs

package runtime

func init() {
	// https://github.com/golang/go/issues/21360
	panic("run go test with -tags fakefs")
}
