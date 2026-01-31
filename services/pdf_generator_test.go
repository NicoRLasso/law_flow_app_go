package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPDFGeneratorOptions(t *testing.T) {
	opts := DefaultPDFOptions()
	assert.Equal(t, "portrait", opts.PageOrientation)
	assert.Equal(t, "letter", opts.PageSize)
	assert.Equal(t, 72, opts.MarginTop)
}
