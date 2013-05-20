// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"espra/config"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func Encode(username, timestamp string, loginID, sessionID int64) string {
	code := fmt.Sprintf("%s|%s|%d|%d", username, timestamp, loginID, sessionID)
	hash := hmac.New(sha256.New, config.SessionKey)
	hash.Write([]byte(code))
	mac := base64.URLEncoding.EncodeToString(hash.Sum(nil))
	return fmt.Sprintf("%s:%s", code, mac)
}

func Decode(auth string) (username string, expires time.Time, loginID int64, sessionID int64, ok bool) {
	s := strings.Split(auth, ":")
	if len(s) != 2 {
		return
	}
	mac, code := s[0], s[1]
	hash := hmac.New(sha256.New, config.SessionKey)
	hash.Write([]byte(code))
	if subtle.ConstantTimeCompare([]byte(base64.URLEncoding.EncodeToString(hash.Sum(nil))), []byte(mac)) != 1 {
		return
	}
	s = strings.SplitN(code, "|", 4)
	if len(s) != 4 {
		return
	}
	username = s[0]
	if username == "" {
		return
	}
	timestamp, err := strconv.ParseInt(s[1], 10, 64)
	if err != nil {
		return
	}
	expires = time.Unix(0, timestamp)
	loginID, err = strconv.ParseInt(s[2], 10, 64)
	if err != nil {
		return
	}
	sessionID, err = strconv.ParseInt(s[3], 10, 64)
	if err != nil {
		return
	}
	ok = true
	return
}

func Info(auth string) (string, bool) {
	username, expires, _, _, ok := Decode(auth)
	if !ok {
		return "", false
	}
	if time.Now().Before(expires) {
		return username, true
	}
	return "", false
}
