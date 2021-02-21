package main

import (
	"flag"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/templates"
)

func main() {
	log.SetLevel(log.DebugLevel)

	var proseFile = flag.String("-p", "proseFiles/max.prose", "source prose file")
	var targetFolder = flag.String("-o", "generatedModules/", "target folder")
	log.Infof("GoProSe: compiling %s", *proseFile)

	parser, err := parser.NewProSeParser(*proseFile)
	if err != nil {
		panic(err)
	}

	err = parser.Parse()
	if err != nil {
		panic(err)
	}

	log.Infof("GoProse: generating code at %s", *targetFolder)
	codeGenerator, err := templates.NewTemplateManager(*targetFolder)
	if err != nil {
		panic(err)
	}

	err = codeGenerator.Render(&templates.ProseProgram{})
	if err != nil {
		panic(err)
	}
}
