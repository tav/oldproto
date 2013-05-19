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

// might be good to write a custom Marshaler for this type to speed up JSON Serialisation=
type Domly []interface{}

type DomlyAttrs map[string]interface{} //string or slice of Domly

type Field struct {
	Key   string
	Value interface{}
	Type  uint8
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

type Index struct {
	Terms []string `datastore:"t"`
}

type Content struct {
	Body       []byte   `datastore:"b,noindex"`
	Data       []*Field `datastore:"d,noindex"`
	Head       []byte   `datastore:"h,noindex"`
	Parents    []string `datastore:"p"`
	RenderType []string `datastore:"r,noindex"`
	Version    uint     `datastore:"v"`
}

type Namespace struct {
}

type Pointer struct {
	Ref string
}
