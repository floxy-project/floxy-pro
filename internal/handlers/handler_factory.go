package handlers

import (
	"strings"

	"github.com/rom8726/floxy-pro"
)

func CreateHandler(
	handlerName string,
	exec string,
	globalTLS *floxy.TLSConfig,
	handlerTLS *floxy.TLSConfig,
	debug bool,
) (floxy.StepHandler, error) {
	exec = strings.TrimSpace(exec)

	if strings.HasPrefix(exec, "http://") || strings.HasPrefix(exec, "https://") {
		tlsConfig := globalTLS
		if handlerTLS != nil {
			tlsConfig = handlerTLS
		}

		return NewHTTPHandler(handlerName, exec, tlsConfig)
	}

	return NewShellHandler(handlerName, exec, debug), nil
}
