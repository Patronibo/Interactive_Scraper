package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"interactive-scraper/internal/service"
)
/*Bu fonksiyon, HTTP üzerinden kullanıcı girişini (login) yöneten bir handler’dır ve Gin web
framework kullanılarak tanımlanmıştır. LoginHandler, gelen JSON isteğini LoginRequest
yapısına bağlar; eğer veri geçersizse 400 Bad Request döner. Ardından,
authService.Login metodu ile kullanıcının kullanıcı adı ve şifresi doğrulanır. Doğrulama
başarılı olursa bir JWT veya benzeri oturum tokeni oluşturulur ve 200 OK ile istemciye
döndürülür; başarısız olursa 401 Unauthorized hatası ile hata mesajı gönderilir. Bu handler,
API’de kullanıcı kimlik doğrulamasının temel noktasını oluşturur.
*/
func LoginHandler(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		token, err := authService.Login(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}
/*Bu fonksiyon, Gin framework üzerinde JWT tabanlı kimlik doğrulama (authentication)
middleware’i olarak çalışır. Gelen HTTP isteğindeki Authorization başlığını kontrol eder;
başlık eksik veya formatı hatalıysa 401 Unauthorized döner ve isteği durdurur. Başlık doğru
formatta ise (Bearer <token>), token authService.ValidateToken ile doğrulanır. Token
geçerli ise içerisindeki kullanıcı bilgisi (Username) context’e eklenir ve istek bir sonraki
handler’a geçer. Bu middleware, API’nin korumalı rotalarında kullanıcı doğrulamasını
merkezi ve güvenli bir şekilde sağlar.
*/
func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := authService.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("username", claims.Username)
		c.Next()
	}
}

