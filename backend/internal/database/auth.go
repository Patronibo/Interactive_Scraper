package database

import (
	"golang.org/x/crypto/bcrypt"
)

/*Bu fonksiyon, verilen bir şifreyi güvenli şekilde hash’leyen yardımcı fonksiyondur.
bcrypt.GenerateFromPassword kullanılarak şifre bcrypt algoritması ile şifrelenir ve geri
döndürülür; işlem sırasında hata oluşursa birlikte iletilir. Bu yöntem, şifreleri veritabanında
düz metin olarak saklamaktan kaçınarak güvenliği artırır.
*/
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

/*bu fonksiyon, verilen bir düz şifre ile hash’lenmiş şifreyi karşılaştıran yardımcı
fonksiyondur. bcrypt.CompareHashAndPassword kullanılarak şifrenin hash ile eşleşip
eşleşmediği kontrol edilir; eşleşiyorsa true, aksi halde false döner. Bu yöntem, kullanıcı
doğrulama sırasında güvenli şifre kontrolü sağlar.
*/
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

