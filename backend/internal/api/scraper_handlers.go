package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"interactive-scraper/internal/scraper"
)

/*Bu fonksiyon, Gin framework üzerinde çalışan ve manuel tüm kaynakları tarama işlemini
başlatan API handler’ıdır. Fonksiyon çağrıldığında önce Tor servisinin hazır olup olmadığı
scraper.CheckTorReadiness() ile kontrol edilir; Tor hazır değilse kullanıcıya 503 Service
unavailable ve ilgili uyarı mesajı döner. Tor hazırsa, scraperService.ScrapeAll() işlemi
arka planda bir goroutine içinde başlatılır ve kullanıcıya işlemin başlatıldığına dair 200 OK
yanıtı ile bildirim gönderilir. Bu yapı, uzun süren scraping işlemlerini bloklamadan arka
planda çalıştırmayı ve kullanıcıya anında geri bildirim vermeyi sağlar.
*/
func TriggerManualScrapeHandler(scraperService *scraper.ScraperService) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("[SCRAPER] TRIGGERED: Manual scrape of all sources requested via API")
		
		
		status, err := scraper.CheckTorReadiness()
		if err != nil || !status.IsReady {
			log.Printf("[SCRAPER] WARNING: Tor not ready when manual scrape triggered: %s", status.Message)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Tor not ready",
				"message": status.Message,
				"status":  "tor_not_ready",
			})
			return
		}
		
		
		go func() {
			log.Println("[SCRAPER] Starting manual scrape of all sources in background...")
			scraperService.ScrapeAll()
			log.Println("[SCRAPER] Finished manual scrape of all sources")
		}()

		c.JSON(http.StatusOK, gin.H{
			"message": "Scraping started in background. Sources will be scraped shortly.",
			"status":  "started",
		})
	}
}

/*Bu fonksiyon, Gin framework üzerinde çalışan ve belirli bir kaynağın manuel olarak
taranmasını başlatan API handler’ıdır. Önce URL’den alınan id parametresi tamsayıya
dönüştürülür; geçersizse 400 Bad Request döner. Eğer aynı kaynak için halihazırda bir
tarama çalışıyorsa, 409 Conflict ile kullanıcıya “Scrape already running” mesajı gönderilir.
Aksi takdirde, scraperService.ScrapeSource işlemi arka planda bir goroutine içinde
başlatılır ve kullanıcıya taramanın başlatıldığına dair 200 OK yanıtı döner. Bu yapı, kaynak
bazlı scraping işlemlerini bloklamadan yönetmeyi ve kullanıcıya hızlı geri bildirim
vermeyi sağlar
*/
func TriggerSourceScrapeHandler(scraperService *scraper.ScraperService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sourceIDStr := c.Param("id")
		sourceID, err := strconv.Atoi(sourceIDStr)
		if err != nil {
			log.Printf("[SCRAPER] ERROR: Invalid source ID in request: %s", sourceIDStr)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid source ID",
				"message": "Source ID must be a valid number",
			})
			return
		}

		log.Printf("[SCRAPER] TRIGGERED: Manual scrape requested for source ID: %d", sourceID)

		
		existingState := scraper.GetScrapeState(sourceID)
		if existingState != nil && existingState.Status == "running" {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "Scrape already running",
				"message": "A scrape is already in progress for this source",
				"status":  "already_running",
				"source_id": sourceID,
			})
			return
		}

		
		go func(id int) {
			log.Printf("[SCRAPER] Starting manual scrape for source ID: %d", id)
			scraperService.ScrapeSource(id)
			log.Printf("[SCRAPER] Finished manual scrape for source ID: %d", id)
		}(sourceID)

		c.JSON(http.StatusOK, gin.H{
			"message":   "Tarama başlatıldı. Kaynak arka planda taranıyor...",
			"status":    "started",
			"source_id": sourceID,
			"note":      "Logları kontrol ederek tarama durumunu görebilirsiniz.",
		})
	}
}

