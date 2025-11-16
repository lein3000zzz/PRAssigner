package mdlwr

import (
	"assignerPR/internal/handlers/apierr"
	"net/http"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

// GetAdminAuthMiddleware - Легковеснее было бы написать свою реализацию, но я решил оставить распространенный пример - он позволяет
// быстрее кастомизировать при изменениях каких-либо
func GetAdminAuthMiddleware(adminSecret string) (*jwt.GinJWTMiddleware, error) {
	return jwt.New(&jwt.GinJWTMiddleware{
		Realm:         "admin zone",
		Key:           []byte(adminSecret),
		Timeout:       time.Hour, // бесполезно, оставил так, как было в примерах с гитхаба
		MaxRefresh:    time.Hour, // тоже не нужно
		Unauthorized:  unauthorizedHandler,
		TokenLookup:   "header: Authorization",
		TokenHeadName: "Bearer",
		Authenticator: func(c *gin.Context) (interface{}, error) {
			return nil, jwt.ErrFailedAuthentication
		},
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			return jwt.MapClaims{}
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			claims := jwt.ExtractClaims(c)
			return claims["role"] == "admin"
		},
	})
}

func unauthorizedHandler(c *gin.Context, _ int, _ string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, apierr.NotFound)
}
