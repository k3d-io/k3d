package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParsePortExposureSpec_PortMatchEnforcement(t *testing.T) {

	r, err := ParsePortExposureSpec("9999", "1111", false)
	require.Nil(t, err)
	require.Equal(t, string(r.Port), "1111/tcp")
	require.Equal(t, string(r.Binding.HostPort), "9999")

	r, err = ParsePortExposureSpec("9999", "1111", true)
	require.Nil(t, err)
	require.Equal(t, string(r.Port), "9999/tcp")
	require.Equal(t, string(r.Binding.HostPort), "9999")

	r, err = ParsePortExposureSpec("random", "1", false)
	require.Nil(t, err)
	require.NotEqual(t, strings.Split(string(r.Port), "/")[0], string(r.Binding.HostPort))

	r, err = ParsePortExposureSpec("random", "", true)
	require.Nil(t, err)
	require.Equal(t, strings.Split(string(r.Port), "/")[0], string(r.Binding.HostPort))
}
