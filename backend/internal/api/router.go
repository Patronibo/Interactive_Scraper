package api

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"interactive-scraper/internal/scraper"
	"interactive-scraper/internal/service"
)

/*Bu fonksiyon, uygulamanın tüm HTTP rotalarını ve middleware’lerini yapılandıran
merkezi router kurulumunu sağlar. Gin framework kullanılarak oluşturulan router,
öncelikle cache kontrol başlıklarını ayarlayan ve CORS politikalarını uygulayan
middleware’lerle donatılır. /api/login rotası ile kullanıcı girişleri yönetilir; /api altındaki
tüm rotalar AuthMiddleware ile korunur. Bu alt grup içerisinde dashboard istatistikleri,
kayıtlar, kategoriler ve kaynak yönetimi gibi API endpoint’leri tanımlanır; scraper ve chat
servislerine ait işlemler de burada erişilebilir hale getirilir. Ayrıca, frontend dosyaları
/static ve / yollarında sunulur ve cache önleme başlıkları eklenir; bilinmeyen rotalar
(NoRoute) da otomatik olarak ana sayfaya yönlendirilir. Bu yapı, hem API hem de frontend
trafiğini merkezi, güvenli ve yönetilebilir şekilde yöneten eksiksiz bir web sunucu altyapısı
sağlar
*/
func SetupRouter(dataService *service.DataService, authService *service.AuthService, scraperService *scraper.ScraperService) *gin.Engine {
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/" || c.Request.URL.Path == "/index.html" {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
		c.Next()
	})

	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	router.Use(cors.New(config))

	router.POST("/api/login", LoginHandler(authService))

	api := router.Group("/api")
	api.Use(AuthMiddleware(authService))
	{
		api.GET("/dashboard/stats", GetDashboardStatsHandler(dataService))
		api.GET("/entries", GetEntriesHandler(dataService))
		api.GET("/entries/:id", GetEntryHandler(dataService))
		api.PUT("/entries/:id/criticality", UpdateCriticalityHandler(dataService))
		api.PUT("/entries/:id/category", UpdateCategoryHandler(dataService))
		api.GET("/categories", GetCategoriesHandler(dataService))
		
		sourceService := service.NewSourceService(dataService.GetDB())
		api.GET("/sources", GetSourcesHandler(sourceService))
		api.GET("/sources/:id", GetSourceHandler(sourceService))
		api.POST("/sources", CreateSourceHandler(sourceService, scraperService))
		api.PUT("/sources/:id", UpdateSourceHandler(sourceService))
		api.DELETE("/sources/:id", DeleteSourceHandler(sourceService))
		
		api.GET("/tor/status", GetTorStatusHandler())
		
		api.POST("/scraper/trigger", TriggerManualScrapeHandler(scraperService))
		api.POST("/sources/:id/scrape", TriggerSourceScrapeHandler(scraperService))
		api.GET("/scraper/status", GetScraperStatusHandler())
		api.GET("/scraper/status/:id", GetSourceScrapeStatusHandler())
		
		api.POST("/chat", ChatHandler())
	}

	router.Static("/static", "./frontend/static")
	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})
	router.StaticFile("/", "./frontend/index.html")
	
	router.NoRoute(func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.File("./frontend/index.html")
	})

	return router
}

