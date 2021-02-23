package parser

import (
	"fmt"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestProSeParser_Parse(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	proseFiles := []string{
		"test.prose",
		"max.prose",
		"tree_coloring.prose",
		"pCover.prose",
		"distributed_reset.prose",
		"routing.prose",
		"pursuer_evader.prose",
	}

	for _, pf := range proseFiles {
		proseFile := fmt.Sprintf("../../proseFiles/%s", pf)
		p, err := NewProSeParser(proseFile, true)
		assert.Nil(t, err)
		if err != nil {
			t.FailNow()
		}

		err = p.Parse()
		assert.Nil(t, err)
		assert.True(t, p.parsed)
	}
}
