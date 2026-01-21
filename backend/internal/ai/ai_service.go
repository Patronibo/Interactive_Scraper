package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

/*Bu kod parçası, uygulama içerisinde yapay zekâ (AI) servisleriyle iletişimi yönetecek olan bir
servis yapısını (struct) tanımlamaktadır. AIService isimli bu yapı, harici bir AI API’sine
yapılacak isteklerin merkezi bir noktadan yönetilmesini amaçlar. Yapı içerisindeki baseURL
alanı, bağlanılacak olan yapay zekâ servisinin ana adresini tutarak tüm HTTP isteklerinin bu
adres üzerinden oluşturulmasını sağlar. httpClient alanı, Go’nun net/http paketine ait
bir http.Client nesnesi olup, isteklerin gönderilmesi, zaman aşımı (timeout) ayarları ve
bağlantı yönetimi gibi işlemleri kontrol eder. enabled değişkeni ise AI servisinin uygulama
içinde aktif olup olmadığını belirlemek için kullanılır; bu sayede servis devre dışı
bırakıldığında uygulama AI çağrıları yapmadan çalışmaya devam edebilir. Genel olarak bu
yapı, yapay zekâ entegrasyonunun daha düzenli, kontrol edilebilir ve genişletilebilir bir
şekilde yönetilmesini sağlar
*/
type AIService struct {
	baseURL    string
	httpClient *http.Client
	enabled    bool
}

/*Bu yapı, yapay zekâ analiz servisine gönderilecek istek verisini temsil eder.
AnalysisRequest struct’ı; analiz edilecek içeriğin başlığını (Title), metnini (Content), ait
olduğu kategoriyi (Category) ve içeriğin önem seviyesini gösteren kritik skorunu
(CriticalityScore) tutar. JSON etiketleri sayesinde bu alanlar, API’ye gönderilirken doğru
formatta serileştirilir.
*/
type AnalysisRequest struct {
	Title            string `json:"title"`
	Content          string `json:"content"`
	Category         string `json:"category"`
	CriticalityScore int    `json:"criticality_score"`
}

/*Bu yapı, yapay zekâ analiz servisinden dönen cevabı temsil eder. AnalysisResponse
struct’ında, AI tarafından üretilen analiz sonucu Analysis alanında tutulur. Error alanı ise
işlem sırasında bir hata oluştuğunda kullanılır; omitempty etiketi sayesinde hata yoksa
JSON çıktısında yer almaz
*/
type AnalysisResponse struct {
	Analysis string `json:"analysis"`
	Error    string `json:"error,omitempty"`
}

/*Bu fonksiyon, AIService yapısını başlatmak ve yapılandırmak için kullanılır. Öncelikle
ortam değişkenlerinden (AI_SERVICE_URL) yapay zekâ servisinin adresi alınır; eğer bu
adres tanımlı değilse AI servisi devre dışı bırakılır (enabled: false). Uygulama Docker
içinde çalışıyorsa ve servis adresi localhost olarak verilmişse, container’ın ana makinedeki
AI servisine erişebilmesi için adres otomatik olarak host.docker.internal şeklinde
güncellenir. Son olarak, 30 saniyelik zaman aşımına sahip bir HTTP istemcisi oluşturularak
AI servisi aktif (enabled: true) olacak şekilde AIService nesnesi döndürülür.
*/
func NewAIService() *AIService {
	baseURL := os.Getenv("AI_SERVICE_URL")
	if baseURL == "" {
		return &AIService{
			enabled: false,
		}
	}

	
	if baseURL == "http://localhost:11434" || baseURL == "http://127.0.0.1:11434" {
		if os.Getenv("DB_HOST") != "" {
			baseURL = "http://host.docker.internal:11434"
		}
	}

	return &AIService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, 
		},
		enabled: true,
	}
}

/*Bu fonksiyon, yapay zekâ servisinin aktif olup olmadığını kontrol etmek için kullanılır.
AIService yapısındaki enabled alanını döndürerek, uygulamanın AI servisine istek
gönderip göndermeyeceğine karar vermesini sağlar.
*/
func (s *AIService) IsEnabled() bool {
	return s.enabled
}

/*Bu fonksiyon, verilen başlık, içerik, kategori ve kritik seviye bilgilerini kullanarak yapay zekâ
servisine analiz isteği göndermek için tasarlanmıştır. Öncelikle AI servisinin aktif olup
olmadığı kontrol edilir; eğer servis kapalıysa sistemin normal akışını bozmamak için hata
üretmeden boş bir sonuç döndürülür. Ardından analiz edilecek veriler AnalysisRequest
yapısına dönüştürülür ve JSON formatına çevrilir. Oluşturulan JSON veri, HTTP POST isteği
ile AI servisinin /analyze endpoint’ine gönderilir. Servisten dönen yanıt başarılıysa, JSON
cevap AnalysisResponse yapısına ayrıştırılır ve varsa hata mesajları kontrol edilir. Son
olarak, AI tarafından üretilen analiz metni elde edilerek çağıran fonksiyona döndürülür;
analiz üretilmemişse bu durum hara olarak değerlendirilmez
*/
func (s *AIService) AnalyzeEntry(title, content, category string, criticalityScore int) (string, error) {
	if !s.enabled {
		return "", nil 
	}

	req := AnalysisRequest{
		Title:            title,
		Content:          content,
		Category:         category,
		CriticalityScore: criticalityScore,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	
	httpReq, err := http.NewRequest("POST", s.baseURL+"/analyze", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("AI service request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(body))
	}

	var aiResp AnalysisResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if aiResp.Error != "" {
		return "", fmt.Errorf("AI service error: %s", aiResp.Error)
	}

	if aiResp.Analysis == "" {
		return "", nil 
	}

	return aiResp.Analysis, nil
}

