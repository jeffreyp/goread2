//go:build !devlogin

package handlers

import "github.com/gin-gonic/gin"

// RegisterDevLogin is a no-op in builds without the devlogin tag, which
// include the default and App Engine builds: they contain no dev login code
// at all. See dev_login.go.
func (ah *AuthHandler) RegisterDevLogin(*gin.RouterGroup) bool {
	return false
}
