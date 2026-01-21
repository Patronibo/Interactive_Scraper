package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"interactive-scraper/internal/service"
)
/*Bu fonksiyon, Gin framework üzerinde çalışan bir dashboard istatistik handler’ıdır ve
DataService aracılığıyla uygulamanın yönetim paneli için gerekli verileri alır.
dataService.GetDashboardStats() çağrısıyla istatistikler çekilir; eğer bir hata oluşursa 500
Internal Server Error ve hata mesajı döndürülür. Başarılı olursa, elde edilen istatistik verileri
JSON formatında 200 OK ile istemciye gönderilir. Bu handler, dashboard verilerinin merkezi
ve güvenli bir şekilde sunulmasını sağlar.
*/
func GetDashboardStatsHandler(dataService *service.DataService) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := dataService.GetDashboardStats()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

/*Bu fonksiyon, Gin framework üzerinde çalışan bir kayıt (entries) listeleme handler’ıdır ve
DataService üzerinden veri tabanındaki kayıtları sayfalı ve filtreli şekilde getirir. Kullanıcı
isteğinden page, pageSize, category ve search parametreleri alınır; sayfa ve sayfa
boyutu için varsayılan değerler atanır. dataService.GetAllEntries çağrısıyla kayıtlar ve
toplam kayıt sayısı çekilir; hata oluşursa 500 Internal Server Error döndürülür. Başarılı
olursa, kayıtlar, toplam kayıt sayısı, sayfa numarası ve sayfa boyutu JSON formatında 200
OK ile istemciye iletilir. Bu yapı, API’de sayfalama ve filtreleme destekli veri sunumu sağlar.
*/
func GetEntriesHandler(dataService *service.DataService) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		category := c.Query("category")
		search := c.Query("search")

		entries, total, err := dataService.GetAllEntries(page, pageSize, category, search)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"entries": entries,
			"total":   total,
			"page":    page,
			"pageSize": pageSize,
		})
	}
}

/*Bu fonksiyon, Gin framework üzerinde çalışan bir tekil kayıt (entry) görüntüleme
handler’ıdır. İstekten URL parametresi olarak alınan id, tamsayıya dönüştürülür;
geçersizse 400 Bad Request döner. Ardından dataService.GetEntryByID ile ilgili kayıt
veritabanından çekilir; kayıt bulunamazsa 404 Not Found döndürülür. Başarılı olursa, kayıt
JSON formatında 200 OK ile istemciye iletilir. Bu handler, API’de belirli bir kaydın güvenli ve
doğru şekilde alınmasını sağlar
*/
func GetEntryHandler(dataService *service.DataService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entry ID"})
			return
		}

		entry, err := dataService.GetEntryByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entry not found"})
			return
		}

		c.JSON(http.StatusOK, entry)
	}
}
/*Bu fonksiyon, Gin framework üzerinde çalışan ve bir kaydın kritiklik (criticality) puanını
güncelleyen handler’dır. URL’den alınan id parametresi tamsayıya dönüştürülür;
geçersizse 400 Bad Request döner. İstek gövdesinde beklenen JSON’dan score değeri
okunur; geçersiz JSON ise aynı şekilde 400 hatası ile geri bildirim yapılır.
dataService.UpdateCriticality metodu çağrılarak ilgili kaydın kritik puanı güncellenir;
hata oluşursa 500 Internal Server Error döner. İşlem başarılı olursa, kullanıcıya “Criticality
updated successfully” mesajı ile 200 OK yanıtı gönderilir. Bu handler, API’de kayıtların
önem seviyesinin güvenli ve kontrollü bir şekilde güncellenmesini sağlar.
*/
func UpdateCriticalityHandler(dataService *service.DataService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entry ID"})
			return
		}

		var req struct {
			Score int `json:"score"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if err := dataService.UpdateCriticality(id, req.Score); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Criticality updated successfully"})
	}
}
/*Bu fonksiyon, Gin framework üzerinde çalışan ve bir kaydın kategori bilgisini güncelleyen
handler’dır. URL’den alınan id parametresi tamsayıya dönüştürülür; geçersizse 400 Bad
Request döner. İstek gövdesinde JSON olarak gönderilen category değeri okunur;
geçersiz JSON ise yine 400 hatası ile geri bildirim yapılır. dataService.UpdateCategory
çağrısı ile ilgili kaydın kategorisi güncellenir; hata oluşursa 500 Internal Server Error döner.
işlem başarılı olursa, kullanıcıya “Category updated successfully” mesajı ile 200 OK yanıtı
gönderilir. Bu handler, API’de kayıtların kategori bilgilerinin güvenli ve kontrollü şekilde
değiştirilmesini sağlar
*/
func UpdateCategoryHandler(dataService *service.DataService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entry ID"})
			return
		}

		var req struct {
			Category string `json:"category"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if err := dataService.UpdateCategory(id, req.Category); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Category updated successfully"})
	}
}
/*Bu fonksiyon, Gin framework üzerinde çalışan ve tüm kayıt kategorilerini listeleyen
handler’dır. dataService.GetCategories çağrısıyla veritabanındaki kategoriler çekilir; hata
oluşursa 500 Internal Server Error döner. Başarılı olursa, kategoriler JSON formatında 200
OK ile istemciye gönderilir. Bu handler, API’de mevcut kategorilerin merkezi ve güvenli bir
şekilde kullanıcıya sunulmasını sağlar.
*/
func GetCategoriesHandler(dataService *service.DataService) gin.HandlerFunc {
	return func(c *gin.Context) {
		categories, err := dataService.GetCategories()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"categories": categories})
	}
}

