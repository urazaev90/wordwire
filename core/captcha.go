// логика, связанная с капчей

package core

import (
	"encoding/json"
	"github.com/dchest/captcha"
	"net/http"
)

// генератор капчи
func GenerateCaptchaHandler(w http.ResponseWriter, r *http.Request) {
	captchaID := captcha.New() // Создаем новую капчу
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"captchaID":  captchaID,                        // Отправляем клиенту ID капчи
		"captchaURL": "/captcha/" + captchaID + ".png", // Ссылка на изображение капчи
	})
}

// проверка капчи
func VerifyCaptcha(captchaID, captchaValue string) bool {
	return captcha.VerifyString(captchaID, captchaValue) // Проверяем правильность введенного значения
}
