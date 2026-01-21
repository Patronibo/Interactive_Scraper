package scraper

import (
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)


type DateParser struct{}

/*Bu fonksiyon, ScraperService içinde yer alan ve bir içeriğin paylaşım tarihini çıkarmayı
amaçlayan metottur. DateParser kullanılarak farklı stratejiler (meta tag’leri, time tag’leri,
yaygın tarih desenleri, ISO8601 formatı, Unix timestamp, göreli tarihler) sırayla denenir.
Eğer herhangi bir strateji geçerli bir tarih bulursa, tarih loglanır ve geri döndürülür. Hiçbir
strateji tarih bulamazsa, fonksiyon nil döner ve sahte tarih oluşturulmaz. Bu yöntem,
scraper’ın içeriklerden doğru ve güvenilir tarih bilgisi elde etmesini sağlar.
*/
func (s *ScraperService) ParseShareDate(rawContent string) *time.Time {
	parser := &DateParser{}
	
	
	strategies := []func(string) *time.Time{
		parser.extractFromMetaTags,
		parser.extractFromTimeTags,
		parser.extractFromCommonPatterns,
		parser.extractFromISO8601,
		parser.extractFromUnixTimestamp,
		parser.extractFromRelativeDates,
	}
	
	for _, strategy := range strategies {
		if date := strategy(rawContent); date != nil {
			log.Printf("[DATE_PARSER] Extracted share date: %s", date.Format(time.RFC3339))
			return date
		}
	}
	
	log.Printf("[DATE_PARSER] No share date found in content")
	return nil 
}

/*Bu fonksiyon, DateParser içinde yer alan ve bir içeriğin HTML meta tag’lerinden
yayınlanma tarihini çıkarmaya çalışan metottur. Fonksiyon, çeşitli yaygın meta tag
formatlarını (article:published_time, og:published_time, date, publishdate,
pubdate, datePublished) regex ile tarar. Eğer bir eşleşme bulunursa, elde edilen tarih
normalizeDate ile standart bir time.Time formatına dönüştürülür ve geri döndürülür.
Hiçbir meta tag eşleşmezse, fonksiyon nil döner. Bu yöntem, scraper’ın sayfaların meta
bilgilerini kullanarak güvenilir tarih bilgisi elde etmesini sağlar.
*/
func (p *DateParser) extractFromMetaTags(content string) *time.Time {
	patterns := []string{
		`(?i)<meta\s+property=["']article:published_time["']\s+content=["']([^"']+)["']`,
		`(?i)<meta\s+property=["']og:published_time["']\s+content=["']([^"']+)["']`,
		`(?i)<meta\s+name=["']date["']\s+content=["']([^"']+)["']`,
		`(?i)<meta\s+name=["']publishdate["']\s+content=["']([^"']+)["']`,
		`(?i)<meta\s+name=["']pubdate["']\s+content=["']([^"']+)["']`,
		`(?i)<meta\s+itemprop=["']datePublished["']\s+content=["']([^"']+)["']`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			if date := p.normalizeDate(matches[1]); date != nil {
				return date
			}
		}
	}
	
	return nil
}

/*Bu fonksiyon, DateParser içinde yer alan ve bir içeriğin HTML <time> tag’lerinden
yayınlanma tarihini çıkarmayı amaçlayan metottur. Farklı <time> tag formatlarını regex ile
kontrol eder (datetime özniteliği, pubdate özniteliği veya tag içeriği). Eşleşme bulunursa,
tarih normalizeDate ile standart time.Time formatına dönüştürülür ve geri döndürülür.
Hiçbir tarih bulunamazsa nil döner. Bu yöntem, scraper’ın sayfa içindeki zaman
tag’lerinden doğru tarih bilgisini güvenilir şekilde elde etmesini sağlar.
*/
func (p *DateParser) extractFromTimeTags(content string) *time.Time {
	patterns := []string{
		`<time\s+datetime=["']([^"']+)["']`,
		`<time\s+pubdate\s+datetime=["']([^"']+)["']`,
		`<time[^>]*>([^<]+)</time>`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			if date := p.normalizeDate(matches[1]); date != nil {
				return date
			}
		}
	}
	
	return nil
}

/*Bu fonksiyon, DateParser içinde yer alan ve bir içeriğin metin içindeki yaygın tarih
desenlerinden yayınlanma tarihini çıkarmayı amaçlayan metottur. ISO 8601 formatları,
ABD ve Avrupa tarih formatları (ör. “Jan 20, 2026” veya “20 January 2026”), sayısal tarih
formatları (DD/MM/YYYY veya MM/DD/YYYY) ve “Published/Posted/Updated” gibi metin
tabanlı tarihler regex ile taranır. Eğer bir eşleşme bulunursa, tarih normalizeDate ile
standart time.Time formatına dönüştürülür ve geri döndürülür. Hiçbir desen eşleşmezse
nil döner. Bu yöntem, scraper’ın sayfa metinlerinden güvenilir tarih bilgisi elde etmesini
sağlar.
*/
func (p *DateParser) extractFromCommonPatterns(content string) *time.Time {
	
	patterns := []string{
		
		`\b(\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2})`,
		`\b(\d{4}-\d{2}-\d{2})`,
		
		
		`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4}`,
		
		
		`\b\d{1,2}\s+(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{4}`,
		
		
		`\b\d{1,2}/\d{1,2}/\d{4}`,
		
		
		`(?i)(?:published|posted|released|updated)[:\s]+([A-Za-z]+\s+\d{1,2},?\s+\d{4})`,
		`(?i)(?:published|posted|released|updated)[:\s]+(\d{1,2}\s+[A-Za-z]+\s+\d{4})`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			if date := p.normalizeDate(matches[1]); date != nil {
				return date
			}
		}
	}
	
	return nil
}

/*Bu fonksiyon, DateParser içinde yer alan ve bir içeriğin ISO 8601 formatındaki tarih ve
saat bilgilerini çıkarmayı amaçlayan metottur. Farklı ISO 8601 biçimleri (tam tarih-zaman,
zaman dilimli veya sadece tarih) regex ile taranır. Eşleşme bulunursa, tarih normalizeDate
ile standart time.Time formatına dönüştürülür ve geri döndürülür. Hiçbir eşleşme
bulunmazsa nil döner. Bu yöntem, scraper’ın standart ISO tarih formatlarından doğru
ve güvenilir tarih bilgisi elde etmesini sağlar.
*/
func (p *DateParser) extractFromISO8601(content string) *time.Time {
	isoPatterns := []string{
		`\b(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})?)`,
		`\b(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})`,
		`\b(\d{4}-\d{2}-\d{2})`,
	}
	
	for _, pattern := range isoPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			if date := p.normalizeDate(matches[1]); date != nil {
				return date
			}
		}
	}
	
	return nil
}

/*Bu fonksiyon, DateParser içinde yer alan ve bir içeriğin Unix timestamp (10 haneli saniye
veya 13 haneli milisaniye) formatındaki tarih bilgisini çıkarmayı amaçlayan metottur.
Regex ile sayısal timestamp’ler bulunur, strconv.ParseInt ile tamsayıya çevrilir ve
gerekirse milisaniyeden saniyeye dönüştürülür. Ardından mantıklı bir tarih aralığında
(2000–2100) olup olmadığı kontrol edilir; geçerliyse time.Unix ile time.Time formatına
dönüştürülüp döndürülür. Geçerli bir tarih bulunamazsa nil döner. Bu yöntem, scraper’ın
sayfalardaki sayısal zaman damgalarından doğru tarih bilgisi elde etmesini sağlar.
*/
func (p *DateParser) extractFromUnixTimestamp(content string) *time.Time {
	
	re := regexp.MustCompile(`\b(1[0-9]{9,12})\b`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		timestamp, err := strconv.ParseInt(matches[1], 10, 64)
		if err == nil {
			
			if timestamp > 1e12 {
				timestamp = timestamp / 1000
			}
			
			if timestamp > 946684800 && timestamp < 4102444800 {
				date := time.Unix(timestamp, 0)
				return &date
			}
		}
	}
	
	return nil
}

/*Bu fonksiyon, DateParser içinde yer alan ve bir içeriğin “X hours/days/weeks/months
ago” gibi göreli tarih ifadelerini çıkarıp gerçek tarih (time.Time) haline getiren metottur.
İçerikteki sayısal değer regex ile bulunur, strconv.Atoi ile tamsayıya çevrilir ve ifade
türüne göre (hour, day, week, month) uygun süre hesaplanarak mevcut zamandan
çıkarılır. Böylece göreli tarih, mutlak bir tarih olarak döndürülür. Eğer içerikte geçerli bir
ifade yoksa fonksiyon nil döner. Bu yöntem, scraper’ın göreli tarihleri doğru şekilde
dönüştürerek analiz yapmasını sağlar.
*/
func (p *DateParser) extractFromRelativeDates(content string) *time.Time {
	now := time.Now()
	
	patterns := map[string]time.Duration{
		`(?i)\b(\d+)\s+hours?\s+ago\b`: 0,
		`(?i)\b(\d+)\s+days?\s+ago\b`:  0,
		`(?i)\b(\d+)\s+weeks?\s+ago\b`: 0,
		`(?i)\b(\d+)\s+months?\s+ago\b`: 0,
	}
	
	for pattern, _ := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			value, err := strconv.Atoi(matches[1])
			if err == nil {
				var duration time.Duration
				if strings.Contains(pattern, "hour") {
					duration = time.Duration(value) * time.Hour
				} else if strings.Contains(pattern, "day") {
					duration = time.Duration(value) * 24 * time.Hour
				} else if strings.Contains(pattern, "week") {
					duration = time.Duration(value) * 7 * 24 * time.Hour
				} else if strings.Contains(pattern, "month") {
					duration = time.Duration(value) * 30 * 24 * time.Hour
				}
				
				date := now.Add(-duration)
				return &date
			}
		}
	}
	
	return nil
}

/*Bu fonksiyon, DateParser içinde yer alan ve çeşitli formatlardaki tarih stringlerini
standart time.Time nesnesine dönüştüren metottur. Önce boşluklar temizlenir, ardından
yaygın tarih ve saat formatları (RFC3339, “YYYY-MM-DD”, “Jan 2, 2006” vb.) tek tek
denenir; başarılı bir parse işleminden sonra, tarih 2000–2100 aralığında ise geri döndürülür.
Ayrıca, gerekirse zaman dilimi kısaltmaları içeren formatlar da kontrol edilir. Eğer hiçbir
format eşleşmezse fonksiyon nil döner. Bu yöntem, scraper’ın farklı kaynaklardan gelen
tarih verilerini tutarlı ve güvenilir bir biçimde normalize etmesini sağlar.
*/
func (p *DateParser) normalizeDate(dateStr string) *time.Time {
	dateStr = strings.TrimSpace(dateStr)
	
	
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"January 2, 2006",
		"Jan 2, 2006",
		"2 January 2006",
		"2 Jan 2006",
		"01/02/2006",
		"2006/01/02",
		"January 2, 2006 3:04 PM",
		"Jan 2, 2006, 3:04 PM",
		"2 Jan 2006 15:04",
		"2006-01-02T15:04:05.000Z",
	}
	
	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			
			if date.Year() >= 2000 && date.Year() <= 2100 {
				return &date
			}
		}
	}
	
	
	if date, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", dateStr); err == nil {
		if date.Year() >= 2000 && date.Year() <= 2100 {
			return &date
		}
	}
	
	return nil
}

