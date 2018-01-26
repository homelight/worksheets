package worksheets

import (
	"strings"

	"github.com/stretchr/testify/require"
)

func (s *Zuite) TestWorksheet_constrainedBy() {
	defs, err := NewDefinitions(strings.NewReader(`worksheet simple {
		1:name text constrained_by { return name == "Alex" || name == "Wilson" }
	}`))
	require.NoError(s.T(), err)

	ws := defs.MustNewWorksheet("simple")

	require.False(s.T(), ws.MustIsSet("name"))
	err = ws.Set("name", NewText("Alice"))
	require.Equal(s.T(), "\"Alice\" not a valid value for constrained field name", err.Error())
	require.False(s.T(), ws.MustIsSet("name"))

	err = ws.Set("name", NewText("Alex"))
	require.NoError(s.T(), err)
	require.True(s.T(), ws.MustIsSet("name"))
	require.Equal(s.T(), `"Alex"`, ws.MustGet("name").String())
}
