package caller

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// Capturer is not CastCapturer {Request(), ..Wait?}
type Capturer interface {
	ToProto() *fm.Clt_Msg_CallResponseRaw

	Request() map[string]interface{}
	Response() map[string]interface{}

	// Check(...) ...
	// FIXME: really not sure that this belongs here:
	CheckFirst() (string, CheckerFunc)
}

// CaptureShower TODO
type CaptureShower interface {
	ShowRequest(func(string, ...interface{})) error
	ShowResponse(func(string, ...interface{})) error
}
