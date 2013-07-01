// Public Domain (-) 2013 The Espra Authors.
// See the Espra UNLICENSE file for details.

package main

import (
	"github.com/tav/golly/log"
	"github.com/tav/golly/optparse"
	"github.com/tav/golly/runtime"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var header = `// DO NOT EDIT.
// Auto-generated API file.
`

type Param struct {
	Name   string           `json:"name"`
	Struct map[string]Param `json:"struct"`
	Type   string           `json:"type"`
}

type Method struct {
	cache    string
	funcname string
	pkgname  string
	pkgpath  string
	Anon     bool    `json:"anon"`
	Doc      string  `json:"doc"`
	In       []Param `json:"in"`
	Name     string  `json:"name"`
	Out      []Param `json:"out"`
}

var methods = []*Method{}

func contains(list []string, item string) bool {
	for _, elem := range list {
		if elem == item {
			return true
		}
	}
	return false
}

func parseDirectory(pkgname, pkgpath, dirpath string, ignore []string) error {
	dir, err := os.Open(dirpath)
	if err != nil {
		return err
	}
	defer dir.Close()
	listing, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}
	if len(listing) == 0 {
		return nil
	}
	files := []string{}
	for _, subpath := range listing {
		path := filepath.Join(dirpath, subpath)
		if contains(ignore, path) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			parseDirectory(subpath, pkgpath+"/"+subpath, path, ignore)
			continue
		}
		if !strings.HasPrefix(subpath, ".") && strings.HasSuffix(subpath, ".go") {
			files = append(files, path)
		}
	}
	if len(files) == 0 {
		return nil
	}
	return parsePackage(pkgname, pkgpath, files)
}

func parsePackage(pkgname, pkgpath string, filenames []string) error {
	fset := token.NewFileSet()
	for _, filename := range filenames {
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			funcdecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if funcdecl.Doc == nil {
				continue
			}
			doc := strings.TrimSpace(funcdecl.Doc.Text())
			if doc == "" {
				continue
			}
			params := funcdecl.Type.Params.List
			if len(params) == 0 {
				continue
			}
			expr, ok := params[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			sel, ok := expr.X.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			if !(sel.X.(*ast.Ident).Name == "rpc" && sel.Sel.Name == "Context") {
				continue
			}
			doclines := strings.Split(doc, "\n")
			def := doclines[0]
			method := &Method{
				pkgname: pkgname,
				pkgpath: pkgpath,
			}
			if strings.HasSuffix(def, ")") {
				splitdef := strings.Split(def, "(")
				if len(splitdef) != 2 {
					continue
				}
				method.Name = splitdef[0]
				splitdef = strings.Split(splitdef[1], ")")
				if len(splitdef) != 2 && splitdef[1] == "" {
					continue
				}
				for _, part := range strings.Split(splitdef[0], ",") {
					switch part = strings.TrimSpace(part); {
					case part == "anon":
						method.Anon = true
					case part == "cache":
						method.cache = "rpc.LongCache"
					case strings.HasPrefix(part, "cache="):
						method.cache = part[6:]
					}
				}
			} else {
				method.Name = def
			}
			if strings.ContainsAny(method.Name, " ,") {
				continue
			}
			method.funcname = funcdecl.Name.String()
			if len(doclines) > 1 {
				method.Doc = strings.TrimSpace(strings.Join(doclines[1:], "\n"))
			}
			methods = append(methods, method)
		}
	}
	return nil
}

func main() {

	opts := optparse.Parser("Usage: genapi [options]", "0.1")

	root := opts.String(
		[]string{"-r", "--root"}, "../src/espra",
		"path to the root package directory for the app", "PATH")

	ignoreList := opts.String(
		[]string{"-i", "--ignore"}, "api.go html.go",
		"space-separated list of files/subdirectories to ignore", "LIST")

	digestFile := opts.String(
		[]string{"-d", "--digest"}, "../etc/app/version.digest",
		"path to write the digest of the API version", "PATH")

	os.Args[0] = "genapi"
	opts.Parse(os.Args)
	log.AddConsoleLogger()

	ignore := strings.Split(*ignoreList, " ")
	for i, path := range ignore {
		ignore[i] = filepath.Join(*root, path)
	}

	pkgpath := strings.Split(*root, "/")
	pkgname := pkgpath[len(pkgpath)-1]

	if err := parseDirectory(pkgname, pkgname, *root, ignore); err != nil {
		runtime.StandardError(err)
	}

	for _, method := range methods {
		log.Info("%#v", method)
	}
	log.Wait()

	_ = *digestFile

}
