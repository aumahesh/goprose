package parser

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestProSeParser_Parse(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	p, err := NewProSeParser("../../proseFiles/max.prose")
	assert.Nil(t, err)
	if err != nil {
		t.FailNow()
	}

	err = p.Parse()
	assert.Nil(t, err)
	assert.True(t, p.parsed)
}
