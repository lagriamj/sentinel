package authn

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type parserStub struct {
	id  string
	err error
}

func (p parserStub) Subject(string) (string, error) {
	return p.id, p.err
}

func TestAuthenticateRequiresBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/private", Authenticate(parserStub{id: "11111111-1111-4111-8111-111111111111"}), func(c *gin.Context) {
		id, ok := UserID(c)
		if !ok {
			t.Fatal("missing user")
		}
		c.String(http.StatusOK, id)
	})

	missing := httptest.NewRequest(http.MethodGet, "/private", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, missing)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", recorder.Code)
	}

	ok := httptest.NewRequest(http.MethodGet, "/private", nil)
	ok.Header.Set("Authorization", "Bearer token")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, ok)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}
