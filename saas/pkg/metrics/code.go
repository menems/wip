package metrics

import (
	"errors"

	"connectrpc.com/connect"
)

// codeOK is code 0 (success). connect does not export a CodeOK constant.
const codeOK connect.Code = 0

// codeOf extracts the connect.Code from err. Returns codeOK (0) when err is
// nil and CodeUnknown when err is non-nil but not a *connect.Error.
func codeOf(err error) connect.Code {
	if err == nil {
		return codeOK
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr.Code()
	}
	return connect.CodeUnknown
}

// codeString returns a human-readable name for c. It returns "ok" for code 0
// because connect.Code.String() returns "code_0" for the zero value.
func codeString(c connect.Code) string {
	if c == codeOK {
		return "ok"
	}
	return c.String()
}
