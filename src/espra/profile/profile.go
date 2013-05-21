// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package profile

import (
	"appengine/datastore"
	"code.google.com/p/go.crypto/scrypt"
	"crypto/subtle"
	"errors"
	"espra/datetime"
	"espra/db"
	"espra/ident"
	"espra/rpc"
	"espra/session"
	"fmt"
	"strings"
	"time"
)

const (
	defaultGravatar = "https://a248.e.akamai.net/assets.github.com%2Fimages%2Fgravatars%2Fgravatar-user-420.png"
)

type LoginInfo struct {
	Client     string `json:"client"`
	Login      string `json:"login"`
	Passphrase string `json:"passphrase"`
	RememberMe bool   `json:"remember_me"`
}

var (
	ErrEmptyLogin      = errors.New("the login parameter cannot be empty")
	ErrEmptyPassphrase = errors.New("the passphrase parameter cannot be empty")
	ErrInvalidLogin    = errors.New("invalid login")
)

func Login(ctx *rpc.Context, req *LoginInfo) (string, error) {
	if req.Login == "" {
		return "", ErrEmptyLogin
	}
	if req.Passphrase == "" {
		return "", ErrEmptyPassphrase
	}
	var loginID int64
	if strings.Contains(req.Login, "@") {
		email := strings.ToLower(req.Login)
		var meta db.LoginEmail
		err := ctx.Get(ctx.StrKey("LE", email, nil), &meta)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				return "", ErrInvalidLogin
			}
			return "", err
		}
		loginID = meta.Login
	} else {
		username, ok := ident.Username(req.Login)
		if !ok {
			return "", ErrInvalidLogin
		}
		var meta db.LoginUsername
		err := ctx.Get(ctx.StrKey("LU", username, nil), &meta)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				return "", ErrInvalidLogin
			}
			return "", err
		}
		loginID = meta.Login
	}
	var login db.Login
	loginKey := ctx.IntKey("L", loginID, nil)
	err := ctx.Get(loginKey, &login)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return "", ErrInvalidLogin
		}
		return "", err
	}
	s := login.Scrypt
	derived, err := scrypt.Key([]byte(req.Passphrase), s.Salt, s.Iterations, s.BlockSize, s.Parallelisation, s.Length)
	if err != nil {
		return "", err
	}
	if subtle.ConstantTimeCompare(derived, login.Passphrase) != 1 {
		return "", ErrInvalidLogin
	}
	now := datetime.UTC()
	sess := &db.Session{
		Client:     req.Client,
		Expires:    datetime.From(now.Add(time.Hour)),
		Initiated:  now,
		RememberMe: req.RememberMe,
	}
	key, err := ctx.Put(ctx.NewKey("S", loginKey), sess)
	if err != nil {
		return "", err
	}
	return session.Encode(login.Username, string(sess.Expires)[1:], loginKey.IntID(), key.IntID()), nil
}

func SessionRenew(ctx *rpc.Context, auth string) (string, bool) {
	return "", false
}

func Signup(ctx *rpc.Context, req *LoginInfo) (bool, string, error) {
	return false, "", nil
}

func SignupDetails(ctx *rpc.Context) {

}

func Gravatar(ctx *rpc.Context, username string, size string) error {
	validatedUsername, ok := ident.Username(username)
	if !ok {
		return fmt.Errorf("invalid username: %s", username)
	}
	imageSize := ctx.ParseUint(size, "invalid size parameter: %s", 150)
	ctx.App.Infof("foo %s", validatedUsername)
	digest := "6cf15f03f4e93f91688b7e6b945c469e"
	ctx.Redirect(fmt.Sprintf("https://secure.gravatar.com/avatar/%s?s=%d&d=%s", digest, imageSize, defaultGravatar))
	return nil
}

func init() {
	rpc.Register("login", Login).Anon()
	rpc.Register("session.renew", SessionRenew)
	rpc.Register("signup", Signup).Anon()
	rpc.Register("signup.details", Signup).Anon()
	rpc.RegisterGet("profile.gravatar", Gravatar).Cache(rpc.LongCache)
}
