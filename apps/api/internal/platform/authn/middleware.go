package authn

import (
	"net/http"
	"strings"

	"sentinel-scanner/apps/api/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

const userIDKey = "auth_user_id"

// Parser reads the subject from an access token.
type Parser interface {
	Subject(raw string) (string, error)
}

// Authenticate requires a bearer access token and stores its subject.
func Authenticate(parser Parser) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := bearerUser(c.GetHeader("Authorization"), parser)
		if !ok {
			httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
			return
		}
		c.Set(userIDKey, userID)
		c.Next()
	}
}

// UserID returns the authenticated operator stored by Authenticate.
func UserID(c *gin.Context) (string, bool) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return "", false
	}
	id, ok := value.(string)
	return id, ok && id != ""
}

func bearerUser(header string, parser Parser) (string, bool) {
	if parser == nil {
		return "", false
	}
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.Contains(token, " ") {
		return "", false
	}
	userID, err := parser.Subject(token)
	if err != nil || userID == "" {
		return "", false
	}
	return userID, true
}
