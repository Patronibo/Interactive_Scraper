package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"interactive-scraper/internal/scraper"
)

/*Bu fonksiyon, Gin framework üzerinde çalışan ve Tor servisinin durumunu sorgulayan API
handler’dır. scraper.CheckTorStatus() ile Tor’un mevcut durumu alınır; hata oluşsa bile
durum nesnesi JSON formatında 200 OK yanıtı ile istemciye iletilir. Bu handler, uygulamada
Tor ağının kullanılabilirliğini ve durumunu güvenli bir şekilde izlemeyi sağlar.
*/
func GetTorStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := scraper.CheckTorStatus()
		if err != nil {
			
			c.JSON(http.StatusOK, status)
			return
		}

		c.JSON(http.StatusOK, status)
	}
}

