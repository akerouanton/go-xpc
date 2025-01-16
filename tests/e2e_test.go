package tests

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// const requirements = `entitlement ["com.apple.application-identifier"] = "9BNSXJN65R.com.foobar.foobar"`

func TestExampleDaemon(t *testing.T) {
	testcases := []struct {
		method string
		want   string
	}{
		{"ping", "ping succeeded"},
		{"add", "3"},
		{"panic", "didn't panic"},
	}

	for _, tc := range testcases {
		t.Run(tc.method, func(t *testing.T) {
			cmd := exec.Command("../example/ExampleDaemon.app/Contents/MacOS/client", "-method", tc.method)
			out, err := cmd.CombinedOutput()

			assert.NoError(t, err)
			assert.Equal(t, tc.want, strings.Trim(string(out), "\n"), "client output mismatch: want %q, got %q", tc.want, string(out))
		})
	}
}
