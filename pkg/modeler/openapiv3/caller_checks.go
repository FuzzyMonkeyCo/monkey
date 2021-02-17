package openapiv3

import (
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

type namedLambda struct {
	name   string
	lambda modeler.CheckerFunc
}

func (m *oa3) callerChecks() []namedLambda {
	return []namedLambda{
		{"connection to server", m.checkConn},
		{"code < 500", m.checkNot5XX},
		//TODO: when decoupling modeler/caller move these to modeler
		{"HTTP code", m.checkHTTPCode},
		//TODO: check media type matches spec here (Content-Type: application/json)
		{"valid JSON response", m.checkValidJSONResponse},
		{"response validates schema", m.checkValidatesJSONSchema},
	}
}

func (m *oa3) checkConn() (s, skipped string, f []string) {
	if err := m.tcap.doErr; err != nil {
		f = append(f, "communication with server could not be established")
		f = append(f, err.Error())
		return
	}
	s = "request sent"
	return
}

func (m *oa3) checkNot5XX() (s, skipped string, f []string) {
	if code := m.tcap.repProto.StatusCode; code >= 500 {
		f = append(f, fmt.Sprintf("server error: '%d'", code))
		return
	}
	s = "no server error"
	return
}

func (m *oa3) checkHTTPCode() (s, skipped string, f []string) {
	if m.tcap.matchedHTTPCode {
		s = "HTTP code checked"
	} else {
		code := m.tcap.repProto.StatusCode
		f = append(f, fmt.Sprintf("unexpected HTTP code '%d'", code))
	}
	return
}

func (m *oa3) checkValidJSONResponse() (s, skipped string, f []string) {
	if len(m.tcap.repProto.Body) == 0 {
		skipped = "response body is empty"
		return
	}

	if m.tcap.repBodyDecodeErr != nil {
		f = append(f, m.tcap.repBodyDecodeErr.Error())
		return
	}

	s = "response is valid JSON"
	return
}

func (m *oa3) checkValidatesJSONSchema() (s, skipped string, f []string) {
	if m.tcap.matchedSID == 0 {
		skipped = "no JSON Schema specified for response"
		return
	}
	if len(m.tcap.repProto.Body) == 0 {
		skipped = "response body is empty"
		return
	}
	if errs := m.vald.Validate(m.tcap.matchedSID, m.tcap.repProto.BodyDecoded); len(errs) != 0 {
		f = errs
		return
	}
	s = "response validates JSON Schema"
	return
}
