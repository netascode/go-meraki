package meraki

import (
	"net/http"

	"github.com/tidwall/gjson"
)

// Res is an API response returned by client requests.
// This is a GJSON result, which offers advanced and safe parsing capabilities.
// https://github.com/tidwall/gjson
type Res struct {
	gjson.Result
	Header http.Header
}
