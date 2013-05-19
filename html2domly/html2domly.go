package main

import (
	"bytes"
	"encoding/json"
	"espra/ui"
	"fmt"
	"github.com/tav/golly/log"
	"github.com/tav/golly/optparse"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	opts := optparse.Parser("Usage: html2domly [options]", "v")

	output_file := opts.String([]string{"-o", "--output"}, "../coffee/templates.coffee", "coffeescript file to compile to", "PATH")
	template_src_dir := opts.String([]string{"-i", "--input"}, "../etc/domly", "templatate source directory", "PATH")
	os.Args[0] = "html2domly"
	opts.Parse(os.Args)

	log.AddConsoleLogger()

	var data []byte
	var err error
	var pretty_str string
	dir, err := os.Open(*template_src_dir)
	defer dir.Close()

	if err != nil {
		log.Error("Error: %v", err)
	}
	out, err := os.Create(*output_file)
	if err != nil {
		log.Error("Error: %v", err)
	}
	defer out.Close()

	names, err := dir.Readdirnames(0)
	if err != nil {
		log.Error("Error: %v", err)
	}

	out.Write([]byte("define 'templates', (exports) ->\n"))
	var basename string
	for _, name := range names {
		if strings.HasSuffix(name, ".html") {
			basename = strings.TrimSuffix(name, ".html")
		} else {
			log.Error("file %v does not end in .html", name)
			continue
		}

		template_path := filepath.Join(*template_src_dir, name)
		data, err = ui.ParseTemplate(template_path)
		if err != nil {
			log.Error("Error: %v", err)
		} else {
			var pretty bytes.Buffer
			err := json.Indent(&pretty, data, ">", "  ")
			if err != nil {
				log.Error("Error: %v", err)
				log.Info("%v", data)
			} else {
				pretty_str = pretty.String()
				log.Info("%v", pretty_str)
			}
			out.Write([]byte(fmt.Sprintf("  exports['%s'] = `%s`\n", basename, data)))
		}

	}
	out.Write([]byte("  return"))

	log.Info("Compiled domly written to: %v", *output_file)

	log.Wait()

}
