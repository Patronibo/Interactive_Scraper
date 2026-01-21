package scraper

import (
	"sync"
	"time"
)

/*Bu yapı (ScrapeState), bir kaynağın tarama durumunu temsil eder ve scraper’ın
ilerlemesini takip etmek için kullanılır. İçinde kaynağın ID ve adı, tarama durumu
(pending, running, completed, failed), taramanın başlangıç ve bitiş zamanları,
bulunan ve veritabanına eklenen entry sayıları ve varsa hata mesajı gibi bilgiler yer alır.
JSON etiketleri sayesinde API cevaplarında kolayca kullanılabilir ve eksik bilgiler
(completed_at veya error) opsiyonel olarak gösterilebilir. Bu yapı, sistemin her kaynağın
scraping sürecini takip etmesini ve raporlamasını sağlar.
*/
type ScrapeState struct {
	SourceID      int       `json:"source_id"`
	SourceName    string    `json:"source_name"`
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	EntriesFound  int       `json:"entries_found"`
	EntriesInserted int     `json:"entries_inserted"`
	Error         string    `json:"error,omitempty"`
}

/*Bu yapı (ScraperStateManager), scraper sisteminde tarama durumlarını yönetmek için 
kullanılır ve eşzamanlı erişime güvenli olacak şekilde tasarlanmıştır. İçinde aktif taramaları 
(activeScrapes) kaynak ID’sine göre tutar ve aynı anda son 50 tarama (recentScrapes) 
bilgisini saklar. sync.RWMutex sayesinde birden fazla goroutine’in okuma-yazma işlemleri 
güvenli bir şekilde yapılabilir. Bu sayede scraper, hem anlık ilerlemeyi hem de geçmiş 
tarama geçmişini eşzamanlı ve güvenilir bir şekilde takip edebilir.
*/
type ScraperStateManager struct {
	mu           sync.RWMutex
	activeScrapes map[int]*ScrapeState 
	recentScrapes []*ScrapeState      
}

var globalStateManager = &ScraperStateManager{
	activeScrapes: make(map[int]*ScrapeState),
	recentScrapes:  make([]*ScrapeState, 0, 50),
}

/*Bu fonksiyon, ScraperStateManager içindeki belirli bir kaynağın (sourceID) aktif tarama 
durumunu güvenli bir şekilde okur. RLock() ile eşzamanlı okuma sırasında kilitlenmeyi 
önler, işlem tamamlandıktan sonra RUnlock() ile kilidi serbest bırakır ve ilgili kaynağın 
ScrapeState nesnesini döndürür. Eğer kaynakta aktif bir tarama yoksa nil döner. Bu 
yöntem, sistemin herhangi bir kaynağın güncel scraping durumunu thread-safe bir 
şekilde sorgulamasını sağlar.
*/
func GetScrapeState(sourceID int) *ScrapeState {
	globalStateManager.mu.RLock()
	defer globalStateManager.mu.RUnlock()
	return globalStateManager.activeScrapes[sourceID]
}

/*Bu fonksiyon, ScraperStateManager içindeki tüm aktif taramaları güvenli bir şekilde okur 
ve bunları bir liste ([]*ScrapeState) olarak döndürür. RLock() ile eşzamanlı okuma 
sırasında kilitlenmeyi önler, defer RUnlock() ile kilidi serbest bırakır. Fonksiyon, 
activeScrapes haritasındaki tüm ScrapeState nesnelerini toplar ve dışarıya bir dilim 
halinde verir, böylece sistemin tüm aktif scraping süreçlerini thread-safe olarak 
görüntülemesi mümkün olur.
*/
func GetAllActiveScrapes() []*ScrapeState {
	globalStateManager.mu.RLock()
	defer globalStateManager.mu.RUnlock()
	
	states := make([]*ScrapeState, 0, len(globalStateManager.activeScrapes))
	for _, state := range globalStateManager.activeScrapes {
		states = append(states, state)
	}
	return states
}

/*Bu fonksiyon, ScraperStateManager içindeki en son tamamlanan taramaları güvenli bir 
şekilde okur ve belirli bir limit kadar ScrapeState nesnesini döndürür. RLock() ile 
eşzamanlı okuma sırasında kilitlenmeyi önler ve defer RUnlock() ile kilidi serbest bırakır. 
Eğer istenen limit, mevcut kayıt sayısından büyükse, yalnızca mevcut kadarını döndürür. 
Fonksiyon, recentScrapes listesinin bir kopyasını oluşturarak geri döndürür, böylece 
dışarıdan yapılan değişiklikler orijinal listeyi etkilemez ve sistemin son tarama geçmişini 
güvenli bir şekilde sorgulamasını sağlar.
*/
func GetRecentScrapes(limit int) []*ScrapeState {
	globalStateManager.mu.RLock()
	defer globalStateManager.mu.RUnlock()
	
	if limit > len(globalStateManager.recentScrapes) {
		limit = len(globalStateManager.recentScrapes)
	}
	
	result := make([]*ScrapeState, limit)
	copy(result, globalStateManager.recentScrapes[:limit])
	return result
}

/*Bu fonksiyon, ScraperStateManager üzerinde yeni bir tarama süreci başlatır ve bunu 
thread-safe şekilde kaydeder. mu.Lock() ile yazma işlemi sırasında kilitlenmeyi sağlar, 
defer mu.Unlock() ile kilidi serbest bırakır. Fonksiyon, verilen sourceID ve sourceName 
ile bir ScrapeState nesnesi oluşturur, Status alanını "running" olarak ayarlar ve 
StartedAt zamanını şu anki zamanla doldurur. Ardından bu yeni durumu activeScrapes 
haritasına ekler, böylece sistem kaynağın aktif tarama durumunu izleyebilir.
*/
func (sm *ScraperStateManager) startScrape(sourceID int, sourceName string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	state := &ScrapeState{
		SourceID:   sourceID,
		SourceName: sourceName,
		Status:     "running",
		StartedAt:  time.Now(),
	}
	sm.activeScrapes[sourceID] = state
}

/*Bu fonksiyon, ScraperStateManager üzerinde bir taramanın tamamlandığını kaydeder ve 
durumu günceller. Önce kilit alınıp thread-safe erişim sağlanır. Fonksiyon, activeScrapes 
içinde verilen sourceID’ye ait durumu bulur; yoksa hiçbir işlem yapmaz. Bulunan durumda 
Status "completed" olarak değiştirilir, CompletedAt zamanını şu anki zamanla doldurur 
ve bulunan/girilen kayıt sayıları (EntriesFound ve EntriesInserted) güncellenir. Daha 
sonra bu tamamlanmış tarama recentScrapes listesine en başa eklenir ve liste 50 öğeyi 
geçerse kırpılır. Son olarak tarama activeScrapes listesinden silinir, böylece sistem artık 
bu kaynağın aktif tarama durumunu izlemez.
*/
func (sm *ScraperStateManager) completeScrape(sourceID int, entriesFound, entriesInserted int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	state, exists := sm.activeScrapes[sourceID]
	if !exists {
		return
	}
	
	now := time.Now()
	state.Status = "completed"
	state.CompletedAt = &now
	state.EntriesFound = entriesFound
	state.EntriesInserted = entriesInserted
	
	
	sm.recentScrapes = append([]*ScrapeState{state}, sm.recentScrapes...)
	if len(sm.recentScrapes) > 50 {
		sm.recentScrapes = sm.recentScrapes[:50]
	}
	
	delete(sm.activeScrapes, sourceID)
}

/*Bu fonksiyon, ScraperStateManager üzerinde bir taramanın başarısız olduğunu kaydeder. 
İşleyişi completeScrape ile çok benzerdir: önce kilit alınır, activeScrapes içinde sourceID 
aranır; yoksa işlem yapılmaz. Bulunan durumda Status "failed" olarak ayarlanır, 
CompletedAt şu anki zamanla doldurulur ve Error alanına başarısızlık nedeni olarak 
err.Error() yazılır. Ardından bu durum, recentScrapes listesine en başa eklenir ve liste 
50 öğeyi geçerse kırpılır. Son olarak tarama activeScrapes listesinden silinir, böylece artık 
aktif olarak takip edilmez.
*/
func (sm *ScraperStateManager) failScrape(sourceID int, err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	state, exists := sm.activeScrapes[sourceID]
	if !exists {
		return
	}
	
	now := time.Now()
	state.Status = "failed"
	state.CompletedAt = &now
	state.Error = err.Error()
	
	
	sm.recentScrapes = append([]*ScrapeState{state}, sm.recentScrapes...)
	if len(sm.recentScrapes) > 50 {
		sm.recentScrapes = sm.recentScrapes[:50]
	}
	
	delete(sm.activeScrapes, sourceID)
}

