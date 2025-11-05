package cleanup

import (
	"net/http"
)

type ExtractUserFn func(req *http.Request) (string, error)
