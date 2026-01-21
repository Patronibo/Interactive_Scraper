package scraper

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"unicode"

	"interactive-scraper/internal/ai"
)

type ScraperService struct {
	db        *sql.DB
	aiService *ai.AIService
}

func NewScraperService(db *sql.DB) *ScraperService {
	return &ScraperService{
		db:        db,
		aiService: ai.NewAIService(),
	}
}

/*Bu Start fonksiyonu, scraper servisinin ana döngüsünü başlatır ve işlem adımlarını şöyle 
işler: Önce log ile servisin başlatıldığı bildirilir. Ardından Tor ağı için hazır olma durumu 
WaitForTorReady ile kontrol edilir; eğer Tor hazır değilse, uyarı mesajları loglanır ancak 
servis yine de çalışmaya devam eder (bu sayede .onion sitelere erişimde hata çıkabilir). Tor 
hazırsa, başarı mesajı loglanır. Daha sonra bir ticker ile her 30 saniyede bir ScrapeAll 
çağrısı yapılacak şekilde periyodik tarama başlatılır. İlk tarama döngüye girmeden önce 
hemen yapılır. Bu fonksiyon bloklayıcıdır, yani çalıştığı sürece scraper sürekli tarama yapar 
ve Tor durumunu her döngüde dikkate alır.
*/
func (s *ScraperService) Start() {
	log.Println("[SCRAPER] Scraper service starting...")

	log.Println("[SCRAPER] Waiting for Tor to become ready...")
	if err := WaitForTorReady(20, 3*time.Second); err != nil {
		log.Printf("[SCRAPER] WARNING: Tor did not become ready: %v", err)
		log.Println("[SCRAPER] WARNING: Scraper will continue but may fail to access .onion sites")
		log.Println("[SCRAPER] WARNING: Tor may still be bootstrapping. Scraper will retry on each scrape.")
	} else {
		log.Println("[SCRAPER] ✓ Tor is ready! Starting scraper...")
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("[SCRAPER] Starting initial scrape...")
	s.ScrapeAll()

	for range ticker.C {
		s.ScrapeAll()
	}
}

/*ScrapeAll fonksiyonu, veritabanındaki tüm kaynakları sırayla taramak için çalışır; önce log 
ile fonksiyonun çağrıldığı belirtilir, ardından veritabanından tüm kaynak id’leri çekilir, hata 
olursa loglanır ve fonksiyon sonlanır, satırlar tek tek okunarak sourceIDs listesine eklenir 
ve okuma sırasında hata olursa uyarı loglanır ama diğer kaynaklar işlenmeye devam eder,
 eğer kaynak bulunamazsa uyarı loglanır ve fonksiyon çıkar, her kaynak için 
 ScrapeSource(sourceID) çağrısı yapılır ve kaynaklar arasında 2 saniye bekleme eklenir, 
 böylece tarama yükü dengelenir ve tüm kaynaklar işlendiğinde tamamlandığı loglanır; 
kısacası veritabanındaki her kaynağı sırayla tarar, hataları loglar ve tarama ilerleyişini kaydeder.
*/
func (s *ScraperService) ScrapeAll() {
	log.Println("[SCRAPER] ScrapeAll() called - fetching sources from database...")
	
	var sourceIDs []int
	rows, err := s.db.Query("SELECT id FROM sources")
	if err != nil {
		log.Printf("[SCRAPER] ERROR: Error fetching sources: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Printf("[SCRAPER] ERROR: Error scanning source ID: %v", err)
			continue
		}
		sourceIDs = append(sourceIDs, id)
	}

	log.Printf("[SCRAPER] Found %d sources to scrape", len(sourceIDs))
	
	if len(sourceIDs) == 0 {
		log.Println("[SCRAPER] WARNING: No sources found in database. Please add sources first.")
		return
	}

	for i, sourceID := range sourceIDs {
		log.Printf("[SCRAPER] Scraping source %d/%d (ID: %d)...", i+1, len(sourceIDs), sourceID)
		s.ScrapeSource(sourceID)
		if i < len(sourceIDs)-1 {
			time.Sleep(2 * time.Second)
		}
	}
	
	log.Printf("[SCRAPER] COMPLETED: ScrapeAll() finished. Processed %d sources.", len(sourceIDs))
}

/*ScrapeSource fonksiyonu, verilen sourceID için kaynak veritabanından çekilip tarama 
sürecini başlatır ve tamamlar; önce log ile fonksiyon çağrısı belirtilir ve defer ile panic 
durumları yakalanır, veritabanından kaynak adı ve URL alınır, hata olursa scrape durumu 
fail olarak kaydedilir, URL boş veya geçersiz formatta ise hata loglanır ve scrape fail 
olur, Tor durumu kontrol edilir, hazır değilse belirli denemelerle beklenir, fetch işlemi 
FetchURLWithRetry ile yapılır, başarısız olursa scrape fail olur, içerik alınırsa 
processFetchedContent ile entry’ler çıkarılır, eğer entry yoksa scrape complete olarak 
kaydedilir, her entry için önce veritabanında var olup olmadığı kontrol edilir, yoksa eklenir 
ve eklenenler sayılır, AI servisi etkinse arka planda analiz talebi gönderilir, işlem 
tamamlandığında tüm entry sayısı ve eklenen entry sayısı loglanır ve scrape durumu 
complete olarak güncellenir; kısacası kaynak doğrulanır, Tor hazırsa fetch yapılır, içerik 
işlenir, entry’ler veritabanına eklenir, hatalar loglanır ve scrape durumu yönetilir
*/
func (s *ScraperService) ScrapeSource(sourceID int) {
	log.Printf("[SCRAPER] ScrapeSource called for source ID: %d", sourceID)
	
	
	var sourceName, sourceURL string
	var scrapeStarted bool
	
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SCRAPER] PANIC recovered in ScrapeSource for source ID %d: %v", sourceID, r)
			if scrapeStarted {
				globalStateManager.failScrape(sourceID, fmt.Errorf("panic: %v", r))
			}
		}
	}()

	err := s.db.QueryRow("SELECT name, url FROM sources WHERE id = $1", sourceID).Scan(&sourceName, &sourceURL)
	if err != nil {
		log.Printf("[SCRAPER] ERROR: Failed to fetch source ID %d from database: %v", sourceID, err)
		globalStateManager.failScrape(sourceID, fmt.Errorf("database error: %v", err))
		return
	}
	
	
	globalStateManager.startScrape(sourceID, sourceName)
	scrapeStarted = true

	log.Printf("[SCRAPER] Source found: ID=%d, Name=%s, URL=%s", sourceID, sourceName, sourceURL)

	if sourceURL == "" {
		err := fmt.Errorf("source URL is empty")
		log.Printf("[SCRAPER] ERROR: Source URL is empty for source '%s' (ID: %d). Skipping.", sourceName, sourceID)
		globalStateManager.failScrape(sourceID, err)
		return
	}

	if !strings.HasPrefix(sourceURL, "http://") && !strings.HasPrefix(sourceURL, "https://") {
		err := fmt.Errorf("invalid URL format: must start with http:// or https://")
		log.Printf("[SCRAPER] ERROR: Invalid URL format for source '%s' (ID: %d). URL must start with http:// or https://. Got: %s", sourceName, sourceID, sourceURL)
		globalStateManager.failScrape(sourceID, err)
		return
	}

	status, err := CheckTorReadiness()
	if err != nil || !status.IsReady {
		torErr := fmt.Errorf("Tor not ready: %s", status.Message)
		log.Printf("[SCRAPER] ERROR: Cannot scrape source ID %d - %v", sourceID, torErr)
		log.Printf("[SCRAPER] Retrying Tor readiness check...")
		
		if waitErr := WaitForTorReady(5, 2*time.Second); waitErr != nil {
			log.Printf("[SCRAPER] ERROR: Tor did not become ready for source ID %d: %v", sourceID, waitErr)
			globalStateManager.failScrape(sourceID, fmt.Errorf("Tor not ready: %v", waitErr))
			return
		}
		log.Printf("[SCRAPER] Tor became ready, continuing scrape for source ID %d", sourceID)
	}

	log.Printf("[SCRAPER] Attempting to fetch from %s via Tor...", sourceURL)
	rawContent, fetchError := FetchURLWithRetry(sourceURL, 3, 5*time.Second)
	if fetchError != nil {
		log.Printf("[SCRAPER] ERROR: Failed to fetch from %s after retries: %v. Skipping this source.", sourceURL, fetchError)
		globalStateManager.failScrape(sourceID, fmt.Errorf("fetch failed: %v", fetchError))
		return
	}

	log.Printf("[SCRAPER] Successfully fetched content from %s (length: %d bytes)", sourceURL, len(rawContent))

	if rawContent == "" {
		err := fmt.Errorf("no content fetched from URL")
		log.Printf("[SCRAPER] ERROR: No content fetched from %s (source ID: %d). Skipping.", sourceURL, sourceID)
		globalStateManager.failScrape(sourceID, err)
		return
	}

	log.Printf("[SCRAPER] Processing fetched content from %s (length: %d bytes)", sourceURL, len(rawContent))
	entries := s.processFetchedContent(sourceName, sourceURL, rawContent)
		log.Printf("[SCRAPER] Processed %d entries from %s", len(entries), sourceURL)
	
	if len(entries) == 0 {
		log.Printf("[SCRAPER] WARNING: No entries extracted from %s after processing (source ID: %d)", sourceURL, sourceID)
		globalStateManager.completeScrape(sourceID, 0, 0)
		return
	}

	log.Printf("[SCRAPER] Processing %d entries for source ID %d", len(entries), sourceID)
	entriesInserted := 0

	for i, entry := range entries {
		log.Printf("[SCRAPER] Processing entry %d/%d: %s", i+1, len(entries), entry.Title)
		
		var exists bool
		err := s.db.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM data_entries WHERE source_id = $1 AND title = $2)
		`, sourceID, entry.Title).Scan(&exists)

		if err != nil {
			log.Printf("[SCRAPER] ERROR: Failed to check entry existence for '%s': %v", entry.Title, err)
			continue
		}

		if exists {
			log.Printf("[SCRAPER] Entry already exists, skipping: %s", entry.Title)
			continue
		}

		var entryID int
		var shareDateValue interface{}
		if entry.ShareDate != nil {
			shareDateValue = *entry.ShareDate
		} else {
			shareDateValue = nil
		}
		
		err = s.db.QueryRow(`
			INSERT INTO data_entries (source_id, title, cleaned_content, share_date, criticality_score, category)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, sourceID, entry.Title, entry.CleanedContent, shareDateValue, entry.CriticalityScore, entry.Category).Scan(&entryID)

		if err != nil {
			log.Printf("[SCRAPER] ERROR: Failed to insert entry '%s': %v", entry.Title, err)
			continue
		}

		entriesInserted++
		log.Printf("[SCRAPER] SUCCESS: New entry inserted - ID: %d, Title: %s, Category: %s, Criticality: %d", 
			entryID, entry.Title, entry.Category, entry.CriticalityScore)

		if s.aiService != nil && s.aiService.IsEnabled() {
			go s.requestAIAnalysis(entryID, entry.Title, entry.CleanedContent, entry.Category, entry.CriticalityScore)
		}
	}

	log.Printf("[SCRAPER] COMPLETED: Source ID %d processed. %d entries inserted, %d entries skipped.", 
		sourceID, 		entriesInserted, len(entries)-entriesInserted)
	
	globalStateManager.completeScrape(sourceID, len(entries), entriesInserted)
}

/*Bu ScrapedEntry yapısı, bir kaynaktan çekilen ve işlenen her bir veri girdisini temsil eder; 
Title entry’nin başlığını, CleanedContent temizlenmiş metin içeriğini, ShareDate 
paylaşım tarihini (varsa) işaret eder, CriticalityScore entry’nin önem derecesini veya 
kritik skorunu, Category ise entry’nin kategorisini tutar. Yani temel olarak, her web 
kaynağından çıkarılan veri bu yapıda paketlenip veritabanına eklenir veya AI analizine gönderilir.
*/
type ScrapedEntry struct {
	Title            string
	CleanedContent   string
	ShareDate        *time.Time
	CriticalityScore int
	Category         string
}

/*Bu processFetchedContent fonksiyonu, bir kaynaktan alınan ham HTML veya metin 
içeriğini işleyip ScrapedEntry dizisi olarak döndürüyor; önce içerik uzunluğu 100 bayttan 
fazla mı diye kontrol ediyor, yeterliyse extractTitle ile başlığı çıkartıyor, cleanContent ile 
temizlenmiş metni elde ediyor, sonra detectCategory ile kategori, 
calculateContentCriticality ile kritik skor belirleniyor ve ParseShareDate ile paylaşım 
tarihi çekiliyor, bunlar ScrapedEntry yapısında paketlenip listeye ekleniyor, eğer içerik çok 
kısa ise entry oluşturulmadan atlanıyor.
*/
func (s *ScraperService) processFetchedContent(sourceName, sourceURL, rawContent string) []ScrapedEntry {
	entries := []ScrapedEntry{}
	
	log.Printf("processFetchedContent: Processing content from %s (length: %d bytes)", sourceURL, len(rawContent))
	
	if len(rawContent) > 100 {
		log.Printf("Content length > 100 bytes, creating entry...")
		title := s.extractTitle(rawContent)
		cleanedContent := s.cleanContent(rawContent)
		
		log.Printf("Extracted title: %s (cleaned content length: %d)", title, len(cleanedContent))
		
		category := s.detectCategory(cleanedContent, title)
		
		criticalityScore := s.calculateContentCriticality(cleanedContent, title, category)
		
		shareDate := s.ParseShareDate(rawContent)
		
		entry := ScrapedEntry{
			Title:            title,
			CleanedContent:   cleanedContent,
			ShareDate:        shareDate,
			CriticalityScore: criticalityScore,
			Category:         category,
		}
		
		entries = append(entries, entry)
		log.Printf("Created entry: %s", title)
	} else {
		log.Printf("Content too short (%d bytes), skipping entry creation", len(rawContent))
	}
	
	return entries
}

/*Bu extractTitle fonksiyonu, içerikten bir başlık çıkarmak için önce <title> etiketini
arıyor; yoksa <h1> etiketi deneniyor; ikisi de yoksa içerik temizlenip (HTML etiketlerinden
arındırılarak) ilk 100 karakter alınarak başlık oluşturuluyor, kelime bütünlüğünü korumak 
için son boşluk noktasına kadar kesiliyor ve “…” ekleniyor; içerik çok kısa veya boşsa 
varsayılan "Content from Source" dönüyor.
*/
func (s *ScraperService) extractTitle(content string) string {
	titleRegex := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	matches := titleRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		if len(title) > 0 && len(title) <= 200 {
			return title
		}
	}

	h1Regex := regexp.MustCompile(`(?i)<h1[^>]*>([^<]+)</h1>`)
	matches = h1Regex.FindStringSubmatch(content)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		if len(title) > 0 && len(title) <= 200 {
			return title
		}
	}

	cleaned := s.cleanContent(content)
	cleaned = strings.TrimSpace(cleaned)
	
	if len(cleaned) == 0 {
		return "Content from Source"
	}

	if len(cleaned) > 100 {
		truncated := cleaned[:100]
		lastSpace := strings.LastIndex(truncated, " ")
		if lastSpace > 50 {
			truncated = truncated[:lastSpace]
		}
		return truncated + "..."
	}

	return cleaned
}

/*Bu cleanContent fonksiyonu, ham HTML içeriğini işlemden geçirip temiz ve okunabilir 
metin hâline getiriyor; önce <script> ve <style> blokları, HTML yorumları ve tüm HTML 
etiketleri kaldırılıyor, ardından HTML entity’leri (&nbsp;, &amp;, &lt; vs.) gerçek 
karakterlere dönüştürülüyor, fazla boşluklar tek boşluğa indirgeniyor ve baştan/sondan 
boşluklar temizleniyor; sadece yazdırılabilir karakterler bırakılıyor ve metin hâlâ çok uzunsa 
(5000 karakteri geçiyorsa) kesilip son boşluk noktasına kadar bırakılarak “…” ekleniyor, 
böylece hem temiz hem makul uzunlukta içerik elde ediliyor.
*/
func (s *ScraperService) cleanContent(rawContent string) string {
	cleaned := rawContent

	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	cleaned = scriptRegex.ReplaceAllString(cleaned, " ")
	
	styleRegex := regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	cleaned = styleRegex.ReplaceAllString(cleaned, " ")

	commentRegex := regexp.MustCompile(`<!--.*?-->`)
	cleaned = commentRegex.ReplaceAllString(cleaned, " ")

	tagRegex := regexp.MustCompile(`<[^>]+>`)
	cleaned = tagRegex.ReplaceAllString(cleaned, " ")

	cleaned = strings.ReplaceAll(cleaned, "&nbsp;", " ")
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", "\"")
	cleaned = strings.ReplaceAll(cleaned, "&#39;", "'")

	whitespaceRegex := regexp.MustCompile(`\s+`)
	cleaned = whitespaceRegex.ReplaceAllString(cleaned, " ")

	cleaned = strings.TrimSpace(cleaned)

	var result strings.Builder
	for _, r := range cleaned {
		if unicode.IsPrint(r) || r == ' ' || r == '\n' || r == '\t' {
			result.WriteRune(r)
		}
	}
	cleaned = result.String()

	cleaned = whitespaceRegex.ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)

	if len(cleaned) > 5000 {
		truncated := cleaned[:5000]
		lastSpace := strings.LastIndex(truncated, " ")
		if lastSpace > 4500 {
			truncated = truncated[:lastSpace]
		}
		cleaned = truncated + "..."
	}

	return cleaned
}

/*Bu detectCategory fonksiyonu, verilen içerik ve başlıktaki anahtar kelimelere bakarak 
içerik kategorisini tahmin ediyor; önce içerik ve başlık küçük harfe çevrilip birleştiriliyor, 
ardından önceden tanımlanmış kategorilere ait anahtar kelimeler içinde kaç tanesinin 
içerikte geçtiği sayılıyor, her kategori için bir puan hesaplanıyor ve en yüksek puanı alan 
kategori seçiliyor; eğer hiçbir anahtar kelime bulunamazsa kategori "Uncategorized" olarak 
atanıyor, aksi halde en yüksek puanlı kategori döndürülüyor, böylece içerik otomatik olarak 
güvenlik, siber saldırı, veri sızıntısı gibi sınıflara ayrılabiliyor.
*/
func (s *ScraperService) detectCategory(content, title string) string {
	contentLower := strings.ToLower(content + " " + title)

	categoryKeywords := map[string][]string{
		"Malware Analysis": {
			"malware", "trojan", "virus", "worm", "ransomware", "spyware",
			"backdoor", "rootkit", "infection", "payload",
		},
		"Data Breach": {
			"breach", "leak", "stolen data", "data exposure", "database dump",
			"credentials leaked", "password dump", "personal information",
		},
		"Vulnerability Disclosure": {
			"vulnerability", "cve-", "exploit", "zero-day", "security flaw",
			"bug", "weakness", "patch", "update required",
		},
		"Cyber Attack": {
			"attack", "hack", "compromised", "intrusion", "unauthorized access",
			"infiltration", "incident", "incursion",
		},
		"Exploit Development": {
			"exploit code", "proof of concept", "poc", "exploit development",
			"metasploit", "payload generator",
		},
		"Network Security": {
			"network", "firewall", "ddos", "dos attack", "traffic",
			"packet", "router", "switch", "infrastructure",
		},
		"Security Research": {
			"research", "analysis", "study", "findings", "paper",
			"whitepaper", "report",
		},
	}

	categoryScores := make(map[string]int)
	for category, keywords := range categoryKeywords {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(contentLower, keyword) {
				score++
			}
		}
		categoryScores[category] = score
	}

	maxScore := 0
	detectedCategory := "Threat Intelligence"
	for category, score := range categoryScores {
		if score > maxScore {
			maxScore = score
			detectedCategory = category
		}
	}

	if maxScore == 0 {
		return "Uncategorized"
	}

	return detectedCategory
}

/*Bu calculateContentCriticality fonksiyonu, içerik ve başlığa bakarak bir “kritiklik” puanı 
(0–100) hesaplıyor; önce kategoriye göre bir temel puan (baseScore) belirleniyor, örneğin 
"Data Breach" için 85, "Cyber Attack" için 90 gibi. Ardından içerikteki yüksek öncelikli 
anahtar kelimeler (critical, zero-day, breach confirmed gibi) bulunursa her biri 5 puan 
artırıyor, düşük öncelikli kelimeler (discussion, informational, historical gibi) bulunursa 
her biri 3 puan düşürüyor. Sonuç olarak puan 0’ın altına düşerse 0, 100’ü geçerse 100’e 
sabitleniyor. Böylece içerik otomatik olarak kritik, yüksek riskli veya daha az önemli 
olarak derecelendirilebiliyor.
*/
func (s *ScraperService) calculateContentCriticality(content, title, category string) int {
	contentLower := strings.ToLower(content + " " + title)
	baseScore := 50

	categoryScores := map[string]int{
		"Malware Analysis":        70,
		"Data Breach":             85,
		"Vulnerability Disclosure": 80,
		"Threat Intelligence":      65,
		"Security Research":        50,
		"Cyber Attack":            90,
		"Exploit Development":      75,
		"Network Security":        60,
		"Uncategorized":           40,
	}

	if score, exists := categoryScores[category]; exists {
		baseScore = score
	}

	highPriorityKeywords := []string{
		"critical", "urgent", "immediate", "severe", "high risk",
		"zero-day", "active exploit", "live attack", "breach confirmed",
		"data leaked", "credentials exposed", "massive breach",
	}

	lowPriorityKeywords := []string{
		"discussion", "forum", "general", "informational", "news",
		"analysis only", "historical", "old",
	}

	highPriorityCount := 0
	for _, keyword := range highPriorityKeywords {
		if strings.Contains(contentLower, keyword) {
			highPriorityCount++
		}
	}

	lowPriorityCount := 0
	for _, keyword := range lowPriorityKeywords {
		if strings.Contains(contentLower, keyword) {
			lowPriorityCount++
		}
	}

	score := baseScore
	score += highPriorityCount * 5
	score -= lowPriorityCount * 3

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

/*Bu calculateCriticality fonksiyonu, yalnızca içerik kategorisine bakarak bir temel 
kritiklik puanı (0–100 arası) döndürüyor; örneğin "Data Breach" için 85, "Cyber Attack" 
için 90 gibi. Eğer kategori listede yoksa (baseScore == 0), varsayılan olarak 50 puan 
veriliyor. Yani bu fonksiyon içerik metnini analiz etmeden sadece kategoriyi temel 
alıyor ve daha basit, hızlı bir kritiklik tahmini sağlıyor.
*/
func (s *ScraperService) calculateCriticality(category string) int {
	categoryScores := map[string]int{
		"Malware Analysis":        70,
		"Data Breach":             85,
		"Vulnerability Disclosure": 80,
		"Threat Intelligence":      65,
		"Security Research":        50,
		"Cyber Attack":            90,
		"Exploit Development":      75,
		"Network Security":        60,
	}

	baseScore := categoryScores[category]
	if baseScore == 0 {
		baseScore = 50
	}

	return baseScore
}

/*Bu requestAIAnalysis fonksiyonu, belirli bir veri girdisi (entryID) için AI servisine analiz 
talebi gönderiyor, eğer analiz başarılı olursa sonucu veritabanındaki ilgili kayda ekliyor; 
hata oluşursa log’a yazıyor ama sistem normal akışına devam ediyor, böylece AI hataları 
scraping sürecini durdurmuyor ve her giriş için opsiyonel bir ek analiz sağlıyor.
*/
func (s *ScraperService) requestAIAnalysis(entryID int, title, content, category string, criticalityScore int) {
	analysis, err := s.aiService.AnalyzeEntry(title, content, category, criticalityScore)
	if err != nil {
		log.Printf("AI analysis failed for entry %d: %v (system continues normally)", entryID, err)
		return
	}

	if analysis == "" {
		return
	}

	_, err = s.db.Exec(`
		UPDATE data_entries 
		SET ai_analysis = $1 
		WHERE id = $2
	`, analysis, entryID)

	if err != nil {
		log.Printf("Error updating AI analysis for entry %d: %v", entryID, err)
	} else {
		log.Printf("AI analysis added for entry %d", entryID)
	}
}

