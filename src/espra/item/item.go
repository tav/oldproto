// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package item

import (
	"appengine/datastore"
	"espra/db"
	"espra/ident"
	"espra/rpc"
	"fmt"
)

type WebLink struct {
	Host string
	URL  string
}

// Returns (domlyAsJSON, references, slashTag, hostURIs)
func parseMsg(message string, terms []string) ([]byte, []string, string, []*WebLink) {
	return []byte("[]"), []string{"foo", "bar"}, "", []*WebLink{}
}

type CreateRequest struct {
	By      string
	Head    string
	Space   string
	Parents []string
}

func Create(ctx *rpc.Context, req *CreateRequest) error {

	item := &db.Item{}
	terms := []string{}

	var ok bool

	if item.Space, ok = ident.Ref(req.Space); !ok {
		return fmt.Errorf("invalid user/space identifier in the 'space' field: %s", req.Space)
	}

	terms = append(terms, db.SpaceTerm+item.Space)

	if item.By, ok = ident.Username(req.By); !ok {
		return fmt.Errorf("invalid username in the 'by' field: %s", req.By)
	}

	terms = append(terms, db.ByTerm+item.By[1:])

	index := &db.Index{}
	item.Domly, index.Terms, item.SlashTag, _ = parseMsg(req.Head, terms)
	_ = datastore.Delete

	return nil

}

func init() {
	rpc.Register("item.create", Create)
}
