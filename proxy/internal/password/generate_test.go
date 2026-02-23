package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	pass, err := Generate(1)
	require.NoError(t, err)
	assert.Len(t, pass, 3)
	assert.Regexp(t, "[a-z]", pass)
	assert.Regexp(t, "[A-Z]", pass)
	assert.Regexp(t, "[0-9]", pass)

	pass, err = Generate(16)
	require.NoError(t, err)
	assert.Len(t, pass, 16)
	assert.Regexp(t, "[a-z]", pass)
	assert.Regexp(t, "[A-Z]", pass)
	assert.Regexp(t, "[0-9]", pass)
}
