package main

import (
	"fmt"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestCompileAndGenerateCode(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	proseFiles := []string{
		"example.prose",
		"max.prose",
		"tree_coloring.prose",
		"pCover.prose",
		"distributed_reset.prose",
		"routing.prose",
		"pursuer_evader.prose",
	}

	templatesFolder = "../templates"

	for _, pf := range proseFiles {
		proseFile := fmt.Sprintf("../proseFiles/%s", pf)
		targetFolder := "../_examples"

		compile(proseFile, targetFolder)
	}
}
