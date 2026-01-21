package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"interactive-scraper/internal/scraper"
)

/*Bu fonksiyon, Gin framework üzerinde çalışan ve scraper servisinin genel durumunu
görüntüleyen API handler’ıdır. scraper.GetAllActiveScrapes() ile şu anda devam eden
tüm taramalar, scraper.GetRecentScrapes(20) ile son 20 tarama kaydı alınır. Elde edilen
veriler JSON formatında 200 OK yanıtı ile istemciye gönderilir. Bu handler, scraping
aktivitelerinin izlenmesini ve yönetim panelinde durumun görüntülenmesini sağlar.
*/
func GetScraperStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		activeScrapes := scraper.GetAllActiveScrapes()
		recentScrapes := scraper.GetRecentScrapes(20)
		
		c.JSON(http.StatusOK, gin.H{
			"active_scrapes": activeScrapes,
			"recent_scrapes": recentScrapes,
		})
	}
}

/*Bu fonksiyon, Gin framework üzerinde çalışan ve belirli bir kaynağın scraping durumunu
görüntüleyen API handler’ıdır. URL’den alınan id parametresi tamsayıya dönüştürülür;
geçersizse 400 Bad Request döner. scraper.GetScrapeState ile kaynağın mevcut tarama
durumu alınır; eğer tarama bulunamazsa 404 Not Found döner. Mevcut durum JSON
formatında 200 OK yanıtı ile istemciye iletilir. Bu handler, her kaynak için scraping
ilerlemesini ve durum bilgisini güvenli bir şekilde izlemeyi sağlar.
*/
func GetSourceScrapeStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		sourceIDStr := c.Param("id")
		sourceID, err := strconv.Atoi(sourceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid source ID",
			})
			return
		}

		state := scraper.GetScrapeState(sourceID)
		if state == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No scrape found for this source",
			})
			return
		}

		c.JSON(http.StatusOK, state)
	}
}

