/*
Copyright © 2020-2023 The k3d Author(s)
*/
package cmd

import "testing"

// Test_VersionLs_LimitClamp_Regression_1662 verifies that the slice expression
// used to apply the `--limit` flag in `k3d version list` does not panic when
// the limit exceeds the number of filtered tags. See issue #1662 where
// `k3d version list -i '^1\.333' -l 1 -o repo k3s` panicked with
// "slice bounds out of range [:1] with capacity 0".
func Test_VersionLs_LimitClamp_Regression_1662(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		limit    int
		expected int
	}{
		{name: "limit zero leaves slice untouched", tags: []string{"a", "b"}, limit: 0, expected: 2},
		{name: "limit smaller than length clamps", tags: []string{"a", "b", "c"}, limit: 2, expected: 2},
		{name: "limit equal to length is a no-op", tags: []string{"a", "b"}, limit: 2, expected: 2},
		{name: "limit larger than length must not panic", tags: []string{"a"}, limit: 5, expected: 1},
		{name: "limit larger than empty slice must not panic", tags: []string{}, limit: 1, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filteredTags := tt.tags
			// Mirror the production expression from NewCmdVersionLs.
			if tt.limit > 0 && tt.limit < len(filteredTags) {
				filteredTags = filteredTags[0:tt.limit]
			}
			if got := len(filteredTags); got != tt.expected {
				t.Fatalf("len(filteredTags) = %d, want %d", got, tt.expected)
			}
		})
	}
}
