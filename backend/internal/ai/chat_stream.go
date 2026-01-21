package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

/*Bu fonksiyon, ChatService üzerinden gelen kullanıcı mesajını akış (stream) halinde yapay
zekâ modeline gönderip, modelin yanıtını anlık olarak bir io.Writer aracılığıyla iletmek
için kullanılır. Önce servis aktif değilse hata döner. Kullanıcının mesajı, siber güvenlik
analisti rolünde kısa ve net yorumlar yapacak şekilde bir prompt içine yerleştirilir.
OllamaGenerateRequest yapısına dönüştürülüp JSON formatına çevrildikten sonra
/api/generate endpoint’ine POST isteği olarak gönderilir ve Stream: true ayarı ile
modelin yanıtı parçalar halinde (chunk) gelir.
Fonksiyon, yanıtı satır satır okuyarak her geçerli JSON parçasını ayrıştırır, hata olup
olmadığını kontrol eder ve geçerli yanıt parçalarını writer aracılığıyla anında gönderir;
böylece istemciye gerçek zamanlı yanıt akışı sağlanır. Yanıtın tamamı fullResponse içinde
biriktirilir ve stream sonunda boş olup olmadığı kontrol edilir. Hatalar ve bağlantı
problemleri uygun şekilde loglanır ve fonksiyon çağırana iletilir. Bu yöntem özellikle uzun
yanıtların bekletilmeden kullanıcıya aktarılması gereken durumlar için idealdir.
*/
func (s *ChatService) ChatStream(userMessage string, writer io.Writer) error {
	if !s.enabled {
		return fmt.Errorf("chat service is not enabled")
	}


	prompt := fmt.Sprintf(`Sen bir siber güvenlik analistisin. Kısa ve net yorum yap (2-3 cümle max).

KURALLAR: Sadece yorumlama yap, karar verme. Sistem zaten kararları verdi.

Mesaj: %s`, userMessage)

	req := OllamaGenerateRequest{
		Model:      s.model,
		Prompt:     prompt,
		Stream:     true,
		NumPredict: 150, 
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to prepare request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", s.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("Chat stream request failed: %v", err)
		return fmt.Errorf("chat service unavailable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Chat service returned status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("chat service error: status %d", resp.StatusCode)
	}

	
	scanner := bufio.NewScanner(resp.Body)
	var fullResponse strings.Builder

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var streamChunk OllamaGenerateResponse
		if err := json.Unmarshal(line, &streamChunk); err != nil {
			continue 
		}

		
		if streamChunk.Error != "" {
			log.Printf("Ollama stream error: %s", streamChunk.Error)
			return fmt.Errorf("ollama error: %s", streamChunk.Error)
		}

		
		if streamChunk.Response != "" {
			fullResponse.WriteString(streamChunk.Response)
			
			
			chunkData := fmt.Sprintf("data: %s\n\n", streamChunk.Response)
			if _, err := writer.Write([]byte(chunkData)); err != nil {
				return fmt.Errorf("failed to write chunk: %v", err)
			}
		}

		
		if streamChunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %v", err)
	}

	// Final response check
	finalResponse := strings.TrimSpace(fullResponse.String())
	if finalResponse == "" {
		return fmt.Errorf("empty response from chat service")
	}

	return nil
}

