// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package db

import (
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
	ByTerm           = "\x00"
	SpaceTerm        = "\x01"
	UserRefTerm      = "\x02"
	WordTerm         = "\x03"
	HashSpaceRefTerm = "\x04"
	LinkRefTerm      = "\x05"
)

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
	Terms []string `datastore:"t"`
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

type Login struct {
	Confirmed  bool         `datastore:"c"`
	Email      string       `datastore:"e"`
	Passphrase []byte       `datastore:"p,noindex"`
	Scrypt     ScryptParams `datastore:"q,noindex"`
	Status     string       `datastore:"s"`
	Username   string       `datastore:"u"`
	Version    int          `datastore:"v"`
}

type LoginEmail struct {
	Email string `datastore:"e"`
	Login int64  `datastore:"l"`
}

type LoginUsername struct {
	Login    int64  `datastore:"l"`
	Username string `datastore:"u"`
}

type Namespace struct {
}

type Pointer struct {
	Ref string
}

type ScryptParams struct {
	BlockSize       int    `datastore:"b,noindex"`
	Iterations      int    `datastore:"i,noindex"`
	Length          int    `datastore:"k,noindex"`
	Salt            []byte `datastore:"s,noindex"`
	Parallelisation int    `datastore:"p,noindex"`
}

type Session struct {
	Client     string    `datastore:"c,noindex"`
	Expires    Timestamp `datastore:"e"`
	Initiated  time.Time `datastore:"i,noindex"`
	RememberMe bool      `datastore:"r,noindex"`
	Salt       []byte    `datastore:"s,noindex"`
}

type User struct {
	FullName string    `datastore:"f,noindex"`
	Gender   string    `datastore:"g"`
	Joined   Timestamp `datastore:"j"`
	Location string    `datastore:"c"`
	Version  int       `datastore:"v" json:"-"`
}
