// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"espra/config"
	"fmt"
	"strconv"
	"strings"
)

var (
	signingKey   = config.SigningKeys[config.CurrentSigningKeyID]
	signingKeyID = fmt.Sprintf("%d", config.CurrentSigningKeyID)
)

func Sign(username string, unixtime, loginID, sessionID int64) string {
	code := fmt.Sprintf("%s|%s|%d|%d|%d", signingKeyID, username, unixtime, loginID, sessionID)
	hash := hmac.New(sha256.New, signingKey)
	hash.Write([]byte(code))
	mac := base64.URLEncoding.EncodeToString(hash.Sum(nil))
	return fmt.Sprintf("%s:%s", code, mac)
}

func Decode(auth string) (username string, unixtime, loginID, sessionID int64, ok bool) {
	s := strings.Split(auth, ":")
	if len(s) != 2 {
		return
	}
	mac, code := s[0], s[1]
	s = strings.SplitN(code, "|", 5)
	if len(s) != 5 {
		return
	}
	keyID, err := strconv.ParseInt(s[0], 10, 64)
	if err != nil || keyID < 0 {
		return
	}
	key, exists := config.SigningKeys[int(keyID)]
	if !exists {
		return
	}
	hash := hmac.New(sha256.New, key)
	hash.Write([]byte(code))
	if subtle.ConstantTimeCompare([]byte(base64.URLEncoding.EncodeToString(hash.Sum(nil))), []byte(mac)) != 1 {
		return
	}
	username = s[1]
	if username == "" {
		return
	}
	unixtime, err = strconv.ParseInt(s[2], 10, 64)
	if err != nil {
		return
	}
	loginID, err = strconv.ParseInt(s[3], 10, 64)
	if err != nil {
		return
	}
	sessionID, err = strconv.ParseInt(s[4], 10, 64)
	if err != nil {
		return
	}
	ok = true
	return
}
