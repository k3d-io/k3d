package util

import (
	"strings"
	"testing"

	"gotest.tools/assert"
)

func Test_ParsePortExposureSpec_PortMatchEnforcement(t *testing.T) {

	r, err := ParsePortExposureSpec("9999", "1111", false)
	if nil != err {
		t.Fail()
	} else {
		assert.Equal(t, string(r.Port), "1111/tcp")
		assert.Equal(t, string(r.Binding.HostPort), "9999")
	}

	r, err = ParsePortExposureSpec("9999", "1111", true)
	if nil != err {
		t.Fail()
	} else {
		assert.Equal(t, string(r.Port), "9999/tcp")
		assert.Equal(t, string(r.Binding.HostPort), "9999")
	}

	r, err = ParsePortExposureSpec("random", "1", false)
	if nil != err {
		t.Fail()
	} else {
		assert.Assert(t, strings.Split(string(r.Port), "/")[0] != string(r.Binding.HostPort))
	}

	r, err = ParsePortExposureSpec("random", "", true)
	if nil != err {
		t.Fail()
	} else {
		assert.Equal(t, strings.Split(string(r.Port), "/")[0], string(r.Binding.HostPort))
	}
}
