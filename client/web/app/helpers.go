package app

import (
	"strconv"

	"github.com/Rhaqim/buckt/pkg/fileutil"
	"github.com/Rhaqim/buckt/pkg/response"
	"github.com/gin-gonic/gin"
)

// requireParam reads a required URL path parameter. When it is missing it writes
// a uniform 400 response and returns ok=false so the caller returns immediately:
//
//	id, ok := requireParam(c, "file_id")
//	if !ok {
//		return
//	}
func requireParam(c *gin.Context, name string) (string, bool) {
	if v := c.Param(name); v != "" {
		return v, true
	}
	c.AbortWithStatusJSON(400, response.Error(name+" is required", ""))
	return "", false
}

// requireForm is requireParam for a required multipart/urlencoded form value.
func requireForm(c *gin.Context, name string) (string, bool) {
	if v := c.PostForm(name); v != "" {
		return v, true
	}
	c.AbortWithStatusJSON(400, response.Error(name+" is required", ""))
	return "", false
}

// abort500 writes a wrapped internal-error response.
func abort500(c *gin.Context, msg string, err error) {
	c.AbortWithStatusJSON(500, response.WrapError(msg, err))
}

// htmxHandled answers an HTMX request with an empty 200 body (so an
// hx-swap="outerHTML" removes the target element) and reports whether it did so.
// Non-HTMX callers fall through to their normal response.
func htmxHandled(c *gin.Context) bool {
	if c.GetHeader("HX-Request") == "true" {
		c.String(200, "")
		return true
	}
	return false
}

// writeFileHeaders sets the standard headers for serving a file body:
// Content-Disposition (sanitised), Content-Type, Content-Length, and the
// nosniff guard. Callers add Cache-Control / CSP as appropriate.
func writeFileHeaders(c *gin.Context, name, contentType string, size int64, disposition string) {
	c.Header("Content-Disposition", fileutil.SafeContentDisposition(disposition, name))
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("X-Content-Type-Options", "nosniff")
}
