package parser

import (
	"testing"
)

func TestTrimString(t *testing.T) {
	uwant := "https://github.com/daneharrigan/hipchat"
	nwant := "hipchat (xmpp)"

	str := "* [hipchat (xmpp)](https://github.com/daneharrigan/hipchat) - A golang package to communicate with HipChat over XMPP.\n"
	name, url, _ := getPackageDetailsFromString(str)

	t.Run("TrimString: name", func(t *testing.T) {
		if nwant != name {
			t.Logf("Name did not match: want = %s, got = %s", nwant, name)
		}
	})
	t.Run("TrimString: url", func(t *testing.T) {
		if uwant != url {
			t.Logf("URL did not match: want = %s, got = %s", uwant, url)
		}
	})
}
