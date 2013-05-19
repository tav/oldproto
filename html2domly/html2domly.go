package main

import (
	"bytes"
	"encoding/json"
	"espra/ui"
	"fmt"
	"github.com/tav/golly/log"
	"github.com/tav/golly/optparse"
	"github.com/tav/golly/runtime"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	opts := optparse.Parser("Usage: html2domly [options]", "v")

	outputFile := opts.String([]string{"-o", "--output"}, "../coffee/templates.coffee", "coffeescript file to compile to", "PATH")
	templatesSrcDir := opts.String([]string{"-i", "--input"}, "../etc/domly", "templatate source directory", "PATH")
	printJSON := opts.Bool([]string{"--print"}, false, "Print the JSON nicely to the output logger")

	os.Args[0] = "html2domly"
	opts.Parse(os.Args)

	log.AddConsoleLogger()

	var (
		data      []byte
		err       error
		prettyStr string
		basename  string
		pretty    bytes.Buffer
	)

	dir, err := os.Open(*templatesSrcDir)
	if err != nil {
		runtime.StandardError(err)
	}
	defer dir.Close()

	out, err := os.Create(*outputFile)
	if err != nil {
		runtime.StandardError(err)
	}
	defer out.Close()

	names, err := dir.Readdirnames(0)
	if err != nil {
		runtime.StandardError(err)
	}

	out.Write([]byte("define 'templates', (exports) ->\n"))

	for _, name := range names {
		if strings.HasSuffix(name, ".html") {
			basename = strings.TrimSuffix(name, ".html")
		} else {
			log.Error("file %v does not end in .html", name)
			continue
		}

		templatePath := filepath.Join(*templatesSrcDir, name)
		data, err = ui.ParseTemplate(templatePath)
		if err != nil {
			log.StandardError(err)
		} else {

			err := json.Indent(&pretty, data, ">", "  ")
			if err != nil {
				log.StandardError(err)
				log.Info("%v", data)
			} else if *printJSON {
				prettyStr = pretty.String()
				pretty.Reset()
				log.Info("%v", prettyStr)
			}
			out.Write([]byte(fmt.Sprintf("  exports['%s'] = `%s`\n", basename, data)))
		}

	}
	out.Write([]byte("  return"))

	outPath, _ := filepath.Abs(*outputFile)
	log.Info("compiled domly written to: %v", outPath)

	log.Wait()

}
