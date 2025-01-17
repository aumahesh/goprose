package main

import (
	"flag"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/intermediate"
	"github.com/aumahesh/goprose/internal/parser"
	"github.com/aumahesh/goprose/internal/templates"
)

const (
	defaultTemplatesFolder = "templates/"
)

var (
	templatesFolder = defaultTemplatesFolder
)

func compile(proseFile, targetFolder string) {

	log.Infof("GoProSe: compiling %s", proseFile)

	parser, err := parser.NewProSeParser(proseFile, false)
	if err != nil {
		log.Fatal(err)
	}

	err = parser.Parse()
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("GoProse: generating intemediate program")
	parsedProgram, err := parser.GetParsedProgram()
	if err != nil {
		log.Fatal(err)
	}
	intermediateProgram, err := intermediate.GenerateIntermediateProgram(parsedProgram)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("GoProse: generating code at %s", targetFolder)
	codeGenerator, err := templates.NewTemplateManager(templatesFolder, targetFolder, intermediateProgram)
	if err != nil {
		log.Fatal(err)
	}

	err = codeGenerator.Render()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetLevel(log.DebugLevel)

	var proseFile = flag.String("p", "proseFiles/max.prose", "source prose file")
	var targetFolder = flag.String("o", "_generatedModules/", "target folder")

	flag.Parse()

	compile(*proseFile, *targetFolder)
}
