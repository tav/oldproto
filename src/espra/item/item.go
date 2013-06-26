// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package item

import (
	"appengine/datastore"
	"espra/db"
	"espra/ident"
	"espra/rpc"
	"espra/ui"
	"fmt"
)

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
	item.Domly, index.Terms, item.SlashTag, _ = ui.parseMsg(req.Head, terms)
	_ = datastore.Delete

	return nil

}

func init() {
	rpc.Register("item.create", Create)
}
