package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"interactive-scraper/internal/ai"
)

var chatService *ai.ChatService

/*Bu kod, program başlatıldığında chatService nesnesini otomatik olarak başlatan bir
init fonksiyonudur. Go’da init() fonksiyonu, main fonksiyonu çalışmadan önce
otomatik olarak çağrılır. Burada ai.NewChatService() çağrısı ile ChatService yapılandırılır
ve uygulama içinde kullanılmak üzere chatService değişkenine atanır. Böylece, uygulama
başladığında sohbet servisi hazır ve kullanıma uygun hale gelir.
*/
func init() {
	chatService = ai.NewChatService()
}

/*Bu fonksiyon, Gin framework üzerinde çalışan bir HTTP chat handler’ıdır ve kullanıcıdan
gelen mesajları AI sohbet servisine iletir. Gelen JSON isteği doğrulanır; Message alanı
zorunludur ve Stream alanı yanıtın akış halinde mi yoksa tek seferde mi alınacağını belirler.
Chat servisi aktif değilse kullanıcıya “servis kullanılamıyor” mesajı döner. Eğer Stream
aktifse, handleStreamingChat çağrılarak yanıt parça parça iletilir; değilse ayrı bir goroutine
içinde chatService.Chat çalıştırılır ve sonuç kanallar aracılığıyla alınır. 25 saniyelik zaman
aşımıyla servis yanıt vermezse kullanıcıya uygun mesaj iletilir. Bu yapı, hem eşzamansız
hem de gerçek zamanlı chat akışını yönetir ve servis hatalarında kullanıcı deneyimini korur.
*/
func ChatHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Message string `json:"message" binding:"required"`
			Stream  bool   `json:"stream"` 
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
			return
		}

		if !chatService.IsEnabled() {
			
			c.JSON(http.StatusOK, gin.H{
				"reply": "Lokal AI chat servisi şu anda kullanılamıyor. Lütfen daha sonra tekrar deneyin.",
			})
			return
		}

		
		if req.Stream {
			handleStreamingChat(c, req.Message)
			return
		}

		
		replyChan := make(chan string, 1)
		errChan := make(chan error, 1)

		go func() {
			reply, err := chatService.Chat(req.Message)
			if err != nil {
				errChan <- err
				return
			}
			replyChan <- reply
		}()

		
		select {
		case reply := <-replyChan:
			c.JSON(http.StatusOK, gin.H{
				"reply": reply,
			})
		case err := <-errChan:
			log.Printf("Chat error: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"reply": "Lokal AI chat servisi şu anda kullanılamıyor. Lütfen daha sonra tekrar deneyin.",
			})
		case <-time.After(25 * time.Second):
			
			c.JSON(http.StatusOK, gin.H{
				"reply": "Lokal AI chat servisi şu anda kullanılamıyor. Lütfen daha sonra tekrar deneyin.",
			})
		}
	}
}

/*Bu fonksiyon, HTTP üzerinden chat yanıtını gerçek zamanlı olarak akış (stream) halinde
iletmek için kullanılır. Fonksiyon, SSE (Server-Sent Events) başlıklarını ayarlayarak istemciyle
sürekli bir veri kanalı oluşturur ve chatService.ChatStream metodunu kullanarak AI
modelinden gelen yanıtları parça parça c.Writer üzerinden gönderir. Eğer akış sırasında
bir hata oluşursa, hata mesajı bir SSE eventi olarak iletilir ve akış sonlandırılır. Yanıt
tamamlandığında ise “done” eventi gönderilerek istemciye akışın bittiği bildirilir; böylece
kullanıcıya anlık ve kesintisiz chat deneyimi sağlanır.
*/
func handleStreamingChat(c *gin.Context, message string) {
	
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") 

	
	err := chatService.ChatStream(message, c.Writer)
	if err != nil {
		log.Printf("Chat stream error: %v", err)
		
		c.SSEvent("error", "Lokal AI chat servisi şu anda kullanılamıyor. Lütfen daha sonra tekrar deneyin.")
		c.Writer.Flush()
		return
	}

	
	c.SSEvent("done", "")
	c.Writer.Flush()
}

