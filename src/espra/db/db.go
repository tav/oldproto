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

//     Kind: A
//     Key: ID
//
type Account struct {
	Banned             bool   `datastore:"b"`
	Confirmed          bool   `datastore:"c"`
	Email              string `datastore:"e"`
	InitialSpace       string `datastore:"i"`
	MailingList        bool   `datastore:"m"`
	NormalisedUsername string `datastore:"n"`
	PendingSignup      User   `datastore:"p,noindex"`
	Status             string `datastore:"s"`
	Username           string `datastore:"u,noindex"`
	Version            int    `datastore:"v"`
}

// AuthToken functions as a secure token that can be passed
// between clients.
//
//     Kind: AT
//     Parent: Login
//     Key: ID
//
type AuthToken struct {
	Created   int64              `datastore:"c,noindex"`
	Expires   datetime.Timestamp `datastore:"e"`
	Info      string             `datastore:"i,noindex"`
	LongLived bool               `datastore:"l"`
	Scopes    string             `datastore:"s,noindex"`
	Type      string             `datastore:"t"`
}

type ReferenceToken struct {
}

// AuthLog stores some basic info about requests so as to
// provide an audit trail for users to detect unauthorised
// access.
//
//     Kind: AL
//     Key: <username>/<acess-token-id>/<ip-addr>/<hash-of-client>
//
type AuthLog struct {
	City        string    `datastore:"c,noindex"`
	CityLatLong string    `datastore:"g,noindex"`
	Country     string    `datastore:"n,noindex"`
	LastSeen    time.Time `datastore:"l,noindex"`
	Region      string    `datastore:"r,noindex"`
	UserAgent   string    `datastore:"u,noindex"`
}

// Might be good to write a custom Marshaler to speed up
// JSON Serialisation.
type Domly []interface{}

type DomlyAttrs map[string]interface{} // string or Domly

type Content struct {
	Body       []byte   `datastore:"b,noindex"`
	Data       []*Field `datastore:"d,noindex"`
	Head       []byte   `datastore:"h,noindex"`
	Parents    []string `datastore:"p"`
	RenderType []string `datastore:"r,noindex"`
	Version    int      `datastore:"v"`
}

type Field struct {
	Key   string
	Value interface{}
	Type  uint8
}

type Index struct {
	Created datetime.Timestamp `datastore:"c"`
	Terms   []string           `datastore:"t"`
}

type Item struct {
	By         string    `datastore:"b,noindex"`
	Created    time.Time `datastore:"c,noindex"`
	Domly      []byte    `datastore:"d,noindex"`
	RenderType []string  `datastore:"r,noindex"`
	Space      string    `datastore:"s,noindex"`
	SlashTag   string    `datastore:"t,noindex"`
}

// Author | Publisher

//     Parent: Login
//     Key: 'p'
//
type LoginAuth struct {
	Passphrase []byte       `datastore:"p,noindex"`
	Scrypt     ScryptParams `datastore:"s,noindex"`
}

//     Kind: EA
//     Key: <normalised-email>
//
type EmailAccount struct {
	Account int64 `datastore:"a"`
}

//     Kind: GA
//     Key: <normalised-github-email>
//
type GithubAccount struct {
	Account int64 `datastore:"a"`
}

// OAuthToken contains a user's tokens for services that
// support OAuth. It is embedded within other structs so as
// to persist authentication with those services.
type OAuthToken struct {
	AccessToken  string    `datastore:"a,noindex"`
	Expiry       time.Time `datastore:"e,noindex"`
	RefreshToken string    `datastore:"r,noindex"`
}

//     Kind: UA
//     Key: <normalised-username>
//
type UsernameAccount struct {
	Account int64 `datastore:"a"`
}

type Namespace struct {
}

type Pointer struct {
	Ref string
}

// ScryptParams stores the parameters used to derive the
// Passphrase field of Account structs.
type ScryptParams struct {
	BlockSize       int    `datastore:"b,noindex"`
	Iterations      int    `datastore:"i,noindex"`
	Length          int    `datastore:"k,noindex"`
	Salt            []byte `datastore:"s,noindex"`
	Parallelisation int    `datastore:"p,noindex"`
}

//     Key: <normalised-username>
//
type User struct {
	FullName string             `datastore:"f,noindex"`
	Gender   string             `datastore:"g"`
	Joined   datetime.Timestamp `datastore:"j"`
	Location string             `datastore:"l"`
	Version  int                `datastore:"v" json:"-"`
}

func init() {
	if len(sentinel) > 1 {
		panic("db: term constants exceed the byte range")
	}
}
