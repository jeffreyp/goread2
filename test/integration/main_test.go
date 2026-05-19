package integration

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	// Set gin to test mode once, before any goroutines start, to avoid the
	// data race caused by parallel tests each calling gin.SetMode concurrently.
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
