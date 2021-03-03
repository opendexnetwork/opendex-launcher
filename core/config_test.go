package core

import (
	"github.com/magiconair/properties/assert"
	"strings"
	"testing"
)

func Test1(t *testing.T) {
	config, err := parseConfig(strings.NewReader(`
[GitHub]
access-token = "abc123"
`))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, config.GitHub.AccessToken, "abc123", "should get access token abc123")
	assert.Equal(t, config.SimnetDir, "")
}
