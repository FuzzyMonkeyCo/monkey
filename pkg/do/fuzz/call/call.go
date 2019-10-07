package call

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg/do/fuzz"
)

// Capturer is not CastCapturer {Request(), ..Wait?}
type Capturer interface {
	Request() map[string]interface{}
	Response() map[string]interface{}

	// FIXME: really not sure that this belongs here:
	CheckFirst() (string, fuzz.CheckerFunc)
}

// CaptureShower TODO
type CaptureShower interface {
	ShowRequest(func(string, ...interface{})) error
	ShowResponse(func(string, ...interface{})) error
}
