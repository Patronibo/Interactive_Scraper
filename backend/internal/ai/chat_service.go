package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

/* Bu yapı, sohbet tabanlı bir yapay zeka modeliyle iletişimi yöneten servis katmanını temsil
eser. ChatService struct'ı; sohbet isteklerinin gönderileceği servis adresini (baseURL),
kullanılacak yapay zeka modelini, HTTP isteklerini yöneten istemciyi (httpClient)
ve servisin aktif olup olmadığını belirleyen durumu (enabled) tutar. Bu sayede sohbet
fonksiyonları merkezi, kontrollü ve gerektiğinde devre dışı bırakılabilir bir yapı üzerinden
yönetilir
*/
type ChatService struct {
	baseURL    string
	model      string
	httpClient *http.Client
	enabled    bool
}

/*Bu yapı, sohbet yapay zekâ servisine gönderilecek istek verisini temsil eder. ChatRequest
struct’ı, kullanıcıdan gelen mesajı Message alanında tutar ve JSON etiketi sayesinde bu
mesaj API’ye doğru formatta iletilir.
*/
type ChatRequest struct {
	Message string `json:"message"`
}

/*Bu yapı, sohbet yapay zekâ servisinden dönen yanıtı temsil eder. ChatResponse struct’ında,
yapay zekânın ürettiği cevap Reply alanında tutulur. Error alanı ise bir hata oluştuğunda
kullanılır ve omitempty etiketi sayesinde hata yoksa JSON çıktısında yer almaz.
*/
type ChatResponse struct {
	Reply string `json:"reply"`
	Error string `json:"error,omitempty"`
}

/*Bu yapı, Ollama tabanlı bir yapay zekâ modelinden metin üretmek için gönderilecek
isteği temsil eder. OllamaGenerateRequest struct’ı, hangi modeli kullanacağını (Model),
üretilecek metnin başlangıç talimatını (Prompt), yanıtın akış halinde gelip gelmeyeceğini
(Stream) ve isteğe bağlı olarak üretilecek maksimum token sayısını (NumPredict) tutar. Bu
sayede modelden özelleştirilmiş ve kontrollü metin çıktıları alınabilir.
*/
type OllamaGenerateRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	Stream    bool   `json:"stream"`
	NumPredict int   `json:"num_predict,omitempty"` 
}

/*Bu yapı, Ollama yapay zekâ modelinden dönen yanıtı temsil eder.
OllamaGenerateResponse struct’ında, modelin ürettiği metin Response alanında tutulur,
Done alanı yanıt üretiminin tamamlanıp tamamlanmadığını gösterir ve Error alanı, bir
hata oluşmuşsa ilgili mesajı içerir; hata yoksa JSON çıktısında yer almaz.
*/
type OllamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

/*Bu fonksiyon, ChatService yapısını başlatmak ve yapılandırmak için kullanılır. Öncelikle
ortam değişkenlerinden (AI_SERVICE_URL) sohbet servisinin adresi ve (AI_MODEL)
kullanılacak yapay zekâ modeli alınır; eğer tanımlı değillerse varsayılan olarak
http://localhost:11434 adresi ve mistral modeli kullanılır. Uygulama Docker içinde
çalışıyorsa ve servis adresi localhost ise, container’ın ana makinedeki servise erişebilmesi
için adres otomatik olarak host.docker.internal şeklinde güncellenir. Son olarak, 30
saniyelik zaman aşımına sahip bir HTTP istemcisi oluşturularak ChatService aktif
(enabled: true) şekilde döndürülür.
*/
func NewChatService() *ChatService {
	baseURL := os.Getenv("AI_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "mistral"
	}

	
	if baseURL == "http://localhost:11434" || baseURL == "http://127.0.0.1:11434" {
		if os.Getenv("DB_HOST") != "" {
				
			baseURL = "http://host.docker.internal:11434"
		}
	}

	return &ChatService{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, 
		},
		enabled: true,
	}
}


func (s *ChatService) IsEnabled() bool {
	return s.enabled
}

/*Bu fonksiyon, ChatService aracılığıyla kullanıcıdan gelen mesajı Ollama tabanlı yapay
zekâ modeline gönderip yanıt almak için kullanılır. Önce servis aktif değilse hata döner.
Kullanıcının mesajı, siber güvenlik analisti rolünde kısa ve net yorumlar yapacak şekilde bir
prompt içine yerleştirilir; modelin yalnızca yorum yapması, karar vermemesi sağlanır. Bu
prompt, OllamaGenerateRequest yapısına dönüştürülüp JSON formatına çevrilir ve servis
adresindeki /api/generate endpoint’ine POST isteği olarak gönderilir. Servisten dönen
yanıt OllamaGenerateResponse yapısına ayrıştırılır, varsa hata kontrol edilir ve yanıtın boş
olup olmadığına bakılır. Son olarak, temizlenmiş yanıt metni çağıran fonksiyona
döndürülür; tüm hata durumlarında ayrıntılı log kaydı tutulur ve uygun hata mesajları
döndürülür
*/
func (s *ChatService) Chat(userMessage string) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("chat service is not enabled")
	}

	prompt := fmt.Sprintf(`Sen bir siber güvenlik analistisin. Kısa ve net yorum yap (2-3 cümle max).

KURALLAR: Sadece yorumlama yap, karar verme. Sistem zaten kararları verdi. Mesaj: %s`, userMessage)

	req := OllamaGenerateRequest{
		Model:      s.model,
		Prompt:     prompt,
		Stream:     false,
		NumPredict: 150,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		log.Printf("Failed to marshal chat request: %v", err)
		return "", fmt.Errorf("failed to prepare request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", s.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create chat request: %v", err)
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("Chat service request failed (network/timeout): %v", err)
		return "", fmt.Errorf("chat service unavailable: %v", err)
	}
		defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Chat service returned non-OK status %d: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("chat service error: status %d", resp.StatusCode)
	}

	var ollamaResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		log.Printf("Failed to decode chat response: %v", err)
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if ollamaResp.Error != "" {
		log.Printf("Ollama returned error: %s", ollamaResp.Error)
		return "", fmt.Errorf("ollama error: %s", ollamaResp.Error)
	}

	reply := strings.TrimSpace(ollamaResp.Response)
	if reply == "" {
		log.Printf("Empty response from chat service")
		return "", fmt.Errorf("empty response from chat service")
	}

	return reply, nil
}

