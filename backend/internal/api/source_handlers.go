package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"interactive-scraper/internal/scraper"
	"interactive-scraper/internal/service"
)

/*Bu fonksiyon, Gin framework üzerinde çalışan ve tüm kaynakları listeleyen API handler’dır.
sourceService.GetAllSources() çağrısıyla veritabanındaki kaynaklar alınır; hata oluşursa
500 Internal Server Error döner. Başarılı olursa, kaynaklar JSON formatında 200 OK yanıtı ile
istemciye iletilir. Bu handler, API’de mevcut kaynakların merkezi ve güvenli bir şekilde
görüntülenmesini sağlar.
*/
func GetSourcesHandler(sourceService *service.SourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sources, err := sourceService.GetAllSources()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"sources": sources})
	}
}

/*Bu fonksiyon, Gin framework üzerinde çalışan ve tek bir kaynağın detaylarını getiren API
handler’dır. URL’den alınan id parametresi tamsayıya dönüştürülür; geçersizse 400 Bad
Request döner. sourceService.GetSourceByID ile ilgili kaynak veritabanından çekilir;
bulunamazsa 404 Not Found döner. Başarılı olursa, kaynak bilgileri JSON formatında 200
OK yanıtı ile istemciye iletilir. Bu handler, API’de belirli bir kaynağın güvenli ve doğru
şekilde sorgulanmasını sağlar.
*/
func GetSourceHandler(sourceService *service.SourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source ID"})
			return
		}

		source, err := sourceService.GetSourceByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source not found"})
			return
		}

		c.JSON(http.StatusOK, source)
	}
}

/*Bu fonksiyon, Gin framework üzerinde çalışan ve yeni bir kaynağı oluşturan API
handler’dır. İstek gövdesinden JSON ile Name ve URL bilgileri alınır; eksik veya geçersizse
400 Bad Request döner. sourceService.CreateSource ile veritabanına yeni kaynak eklenir;
hata oluşursa 500 Internal Server Error döner. Kaynak başarıyla eklendikten sonra, arka
planda bir goroutine ile otomatik scraping işlemi başlatılır. Bu süreçte Tor servisinin hazır
olması beklenir; hazır değilse sonraki periyodik taramada işleme alınır. Kullanıcıya ise
oluşturulan kaynak JSON formatında 200 OK yanıtıyla iletilir. Bu handler, API’de kaynak
ekleme ve eklenen kaynağın otomatik olarak taranmasını güvenli ve asenkron şekilde
sağlar
*/
func CreateSourceHandler(sourceService *service.SourceService, scraperService *scraper.ScraperService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
			URL  string `json:"url" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		source, err := sourceService.CreateSource(req.Name, req.URL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		
		log.Printf("[SCRAPER] TRIGGERED: Source '%s' added (ID: %d), URL: %s", source.Name, source.ID, source.URL)
		go func(sourceID int, sourceName, sourceURL string) {
			log.Printf("[SCRAPER] Waiting for Tor to be ready for auto-scrape of source ID %d...", sourceID)
			if err := scraper.WaitForTorReady(10, 3*time.Second); err != nil {
				log.Printf("[SCRAPER] WARNING: Tor did not become ready for auto-scrape of source ID %d: %v", sourceID, err)
				log.Printf("[SCRAPER] Source will be scraped on next periodic scrape cycle")
				return
			}
			
			log.Printf("[SCRAPER] Tor is ready. Starting automatic scrape for newly added source - ID: %d, Name: %s, URL: %s", sourceID, sourceName, sourceURL)
			scraperService.ScrapeSource(sourceID)
			log.Printf("[SCRAPER] Finished automatic scrape for source ID: %d", sourceID)
		}(source.ID, source.Name, source.URL)

		c.JSON(http.StatusOK, source)
	}
}

/*Bu fonksiyon, Gin framework üzerinde çalışan ve var olan bir kaynağın bilgilerini
güncelleyen API handler’dır. URL’den alınan id parametresi tamsayıya dönüştürülür;
geçersizse 400 Bad Request döner. İstek gövdesinden JSON ile Name ve URL bilgileri alınır;
eksik veya geçersizse yine 400 hatası döner. sourceService.UpdateSource ile ilgili kaynak
veritabanında güncellenir; hata oluşursa 500 Internal Server Error döner. Başarılı olursa,
kullanıcıya “Source updated successfully” mesajı ile 200 OK yanıtı gönderilir. Bu handler,
API’de kaynakların güvenli ve kontrollü bir şekilde güncellenmesini sağlar.
*/
func UpdateSourceHandler(sourceService *service.SourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source ID"})
			return
		}

		var req struct {
			Name string `json:"name" binding:"required"`
			URL  string `json:"url" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if err := sourceService.UpdateSource(id, req.Name, req.URL); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Source updated successfully"})
	}
}

/*Bu fonksiyon, Gin framework üzerinde çalışan ve belirli bir kaynağı veritabanından silen
API handler’dır. URL’den alınan id parametresi tamsayıya çevrilir; geçersizse 400 Bad
Request döner. sourceService.DeleteSource metodu ile ilgili kaynak silinir; hata oluşursa
500 Internal Server Error ile geri bildirim yapılır. İşlem başarılı olursa, kullanıcıya “Source
deleted successfully” mesajı ile 200 OK yanıtı gönderilir. Bu handler, API’de kaynakların
güvenli ve kontrollü bir şekilde silinmesini sağlar.
*/
func DeleteSourceHandler(sourceService *service.SourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source ID"})
			return
		}

		if err := sourceService.DeleteSource(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Source deleted successfully"})
	}
}

