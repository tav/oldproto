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

func parseMsg(message string, terms []string) (db.Domly, []string, string) {
}

func Create(ctx *rpc.Context, space, by, head string, parents []string) error {
	item := &db.Item{}
	var ok bool
	terms := []string{}
	if item.Space, ok = ident.Ref(to); !ok {
		return fmt.Errorf("invalid user/space identifier in the 'space' field: %s", to)
	}
	terms = append(terms, SpaceTerm+item.Space)
	if item.By, ok = ident.User(by); !ok {
		return fmt.Errorf("invalid user identifier in the 'by' field: %s", by)
	}
	terms = append(terms, ByTerm+item.By[1:])
	item.Domly, index.Terms, item.SlashTag = parseMsg(head, terms)
}

func init() {
	rpc.Register("item.create", Create)
}
