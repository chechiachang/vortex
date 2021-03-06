package web

import (
	"net/http"

	restful "github.com/emicklei/go-restful"
	"github.com/linkernetworks/vortex/src/serviceprovider"
)

// Context is the struct to combine the restful message with our own serviceProvider
type Context struct {
	ServiceProvider *serviceprovider.Container
	Request         *restful.Request
	Response        *restful.Response
}

// NativeContext is the struct to combine the native http message with our own serviceProvider
type NativeContext struct {
	ServiceProvider *serviceprovider.Container
	Request         *http.Request
	Response        http.ResponseWriter
}
