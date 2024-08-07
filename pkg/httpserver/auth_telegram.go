package httpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type AuthTelegramConfig struct {
	Token     string
	TrustedId int64
}

func isAuthorizedByTelegram(request *http.Request, cfg AuthTelegramConfig) bool {
	cookie, err := request.Cookie("telegram_data")
	if err != nil {
		return false
	}
	telegramData := cookie.Value
	if len(telegramData) == 0 {
		return false
	}
	requestTelegramUserData, valid := authTelegram(cfg.Token, telegramData)
	if !valid || cfg.TrustedId != requestTelegramUserData.Id {
		return false
	}
	return true
}

type telegramUserData struct {
	Id              int64  `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	LanguageCode    string `json:"language_code"`
	IsPremium       bool   `json:"is_premium"`
	AllowsWriteToPm bool   `json:"allows_write_to_pm"`
}

func (d *telegramUserData) String() string {
	res, err := json.Marshal(d)
	if err != nil {
		return strconv.Itoa(int(d.Id))
	}
	return string(res)
}

func authTelegram(telegramToken string, requestTelegramData string) (*telegramUserData, bool) {
	teleData, err := url.ParseQuery(requestTelegramData)
	if err != nil {
		return nil, false
	}
	hash, user := teleData.Get("hash"), teleData.Get("user")
	if len(hash) == 0 || len(user) == 0 {
		return nil, false
	}
	strs := []string{}
	for k := range teleData {
		if k == "hash" {
			continue
		}
		strs = append(strs, k+"="+teleData.Get(k))
	}
	sort.Strings(strs)
	dataCheckString := strings.Join(strs, "\n")

	calculatedHash := hex.EncodeToString(hmacSha256(dataCheckString, hmacSha256(telegramToken, []byte("WebAppData"))))
	if calculatedHash != hash {
		return nil, false
	}
	var userData telegramUserData
	if err := json.Unmarshal([]byte(user), &userData); err != nil {
		return nil, false
	}
	if userData.Id == 0 {
		return nil, false
	}
	return &userData, true

}

func hmacSha256(data string, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
