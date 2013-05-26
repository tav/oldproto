// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package db

import (
	"espra/datetime"
	"time"
)

const (
	NullType uint8 = iota
	BoolType
	DateType
	FileType
	FloatType
	GeoType
	ListType
	NumberType
	ObjectType
	RefType
	StringType
	TextType
	UnitType
)

const (
	ByTerm string = string(iota)
	SpaceTerm
	UserRefTerm
	WordTerm
	HashSpaceRefTerm
	LinkRefTerm
	sentinel
)

type AccessToken struct {
}

// Account holds the core meta information about a user's
// account.
//
//     Key: ID
//
type Account struct {
	Confirmed    bool   `datastore:"c"`
	Email        string `datastore:"e"` /* Not normalised! Explicitly as provided. */
	InitialSpace string `datastore:"i"`
	MailOut      bool   `datastore:"m"`
	Package      string `datastore:"p"`
	Suspended    bool   `datastore:"s"`
	TwoFactor    bool   `datastore:"t"`
	Username     string `datastore:"u"`
	Version      int    `datastore:"v"`
}

// AccountLogin stores an overview of the current state of
// the parent Account and the derived key of the passphrase
// after running it through scrypt with the associated
// parameters.
//
//     Parent: Account
//     Key: l
//
type AccountLogin struct {
	DerivedKey []byte       `datastore:"d,noindex"`
	Params     ScryptParams `datastore:"p,noindex"`
	Status     int          `datastore:"s,noindex"`
	Username   string       `datastore:"u,noindex"` /* Not normalised! Explicitly as provided. */
	Version    int          `datastore:"v"`
}

// ClientLog stores some basic info about requests so as to
// provide an audit trail for users to detect unauthorised
// access.
//
//     Key: <account-id>/<client-token-id>/<ip-addr>/<hash-of-user-agent>
//
type ClientLog struct {
	City        string    `datastore:"c,noindex"`
	CityLatLong string    `datastore:"g,noindex"`
	Country     string    `datastore:"n,noindex"`
	LastSeen    time.Time `datastore:"l,noindex"`
	Region      string    `datastore:"r,noindex"`
	UserAgent   string    `datastore:"u,noindex"`
}

// ClientToken functions as a secure token to identify
// authorised clients.
//
//     Parent: Account
//     Key: ID
//
type ClientToken struct {
	Created   time.Time          `datastore:"c,noindex"`
	Expires   datetime.Timestamp `datastore:"e"`
	Info      string             `datastore:"i,noindex"`
	LongLived bool               `datastore:"l"`
	Scopes    string             `datastore:"s,noindex"`
	Type      string             `datastore:"t"`
}

// TODO(salfield): It might be a good idea to write a custom
// Marshaler to speed up JSON encoding of Domly structs.
type Domly []interface{}

type DomlyAttrs map[string]interface{} // string or Domly

type Content struct {
	Body       []byte   `datastore:"b,noindex"`
	Data       []*Field `datastore:"d,noindex"`
	Head       []byte   `datastore:"h,noindex"`
	Parents    []string `datastore:"p,noindex"`
	RenderType []string `datastore:"r,noindex"`
	Version    int      `datastore:"v"`
}

// EmailAccount links an email address to a specific Account.
//
//     Key: <normalised-email>
//
type EmailAccount struct {
	Account int64 `datastore:"a"`
}

type Field struct {
	Key   string
	Value interface{}
	Type  uint8
}

// GithubAccount links a GitHub user account with one of
// ours.
//
//     Key: <normalised-github-email>
//
type GithubAccount struct {
	Account int64      `datastore:"a"`
	OAuth   OAuthToken `datastore:"o,noindex"`
}

// Index stores indexed terms regarding an Item.
//
//     Parent: Item
//     Key: 'i'
//
type Index struct {
	Created datetime.Timestamp `datastore:"c"`
	Terms   []string           `datastore:"t"`
}

//
//     Parent: User
//     Key:
//
type Item struct {
	By         string    `datastore:"b,noindex"`
	Created    time.Time `datastore:"c,noindex"`
	Domly      []byte    `datastore:"d,noindex"`
	Parents    []string  `datastore:"p,noindex"`
	RenderType []string  `datastore:"r,noindex"`
	Space      string    `datastore:"s,noindex"`
	SlashTag   string    `datastore:"t,noindex"`
}

// Author | Publisher

// OAuthToken contains a user's tokens for services that
// support OAuth. It is embedded within other structs so as
// to persist authentication with those services.
type OAuthToken struct {
	AccessToken  string    `datastore:"a,noindex"`
	Expiry       time.Time `datastore:"e,noindex"`
	RefreshToken string    `datastore:"r,noindex"`
}

// Pointer maps a link ref to either an Item ref or to
// another link ref.
//
//     Parent: Space || User
//     Key: <path>
//
type Pointer struct {
	Ref string
}

// ScryptParams stores the parameters used to derive a key
// from a passphrase.
type ScryptParams struct {
	BlockSize       int    `datastore:"b,noindex"`
	Iterations      int    `datastore:"i,noindex"`
	Length          int    `datastore:"l,noindex"`
	Salt            []byte `datastore:"s,noindex"`
	Parallelisation int    `datastore:"p,noindex"`
}

// User stores basic info about a user and acts as the root
// entity for all Item writes.
//
//     Key: <normalised-username>
//
type User struct {
	FullName string             `datastore:"f,noindex" json:"fullname"`
	Gender   string             `datastore:"g" json:"gender"`
	Joined   datetime.Timestamp `datastore:"j" json:"joined<d>"`
	Location string             `datastore:"l" json:"location"`
	Status   int                `datastore:"s" json:"-"`
	Username string             `datastore:"u" json:"username"` /* Not normalised! Explicitly as provided. */
	Version  int                `datastore:"v" json:"-"`
}

// UserIndex stores indexed terms regarding a User.
//
//     Parent: User
//     Key: 'i'
//
type UserIndex struct {
	Terms []string `datastore:"t"`
}

// UsernameAccount links a username to a specific Account.
//
//     Key: <normalised-username>
//
type UsernameAccount struct {
	Account int64 `datastore:"a"`
}

// type AccountChange struct {
// }
//
// type AccountSettings struct {
// }
//
// type Namespace struct {
// }
//
// RefBookmark keeps track of which refs that a user wants to
// automatically load for a given SavedSession.
//
//     Parent: SavedSession
//     Key: ID
//
// type RefBookmark struct {
// 	AutoJoin bool   `datastore:"a"`
// 	Options  []byte `datastore:"o,noindex"`
// 	Ref      string `datastore:"r"`
// }
//
// SavedSession lets users save some state they can reload
// at a later time.
//
//     Parent: Account
//     Key: ID
//
// type SavedSession struct {
// 	Created time.Time `datastore:"c,noindex"`
// 	Default bool      `datastore:"d"`  Invariant: only one can be default at any given time
// 	Name    string    `datastore:"n,noindex"`
// 	Updated time.Time `datastore:"u"`
// }

// The package initialiser ensures that Term constants
// longer than one byte aren't accidentally defined.
func init() {
	if len(sentinel) > 1 {
		panic("db: term constants exceed the byte range")
	}
}
