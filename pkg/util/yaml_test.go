package util

import (
	"bytes"
	"strings"
	"testing"

	"gotest.tools/assert"
)

type DummyContext struct {
	Name string
}

type DummyContextWithTag struct {
	Name string `json:"newName"`
}

func TestSplitYAML(t *testing.T) {
	testSets := map[string]struct {
		document string
		expected []string
	}{
		"single": {
			document: `name: clusterA`,
			expected: []string{
				`name: clusterA`,
			},
		},
		"multiple": {
			document: `name: clusterA
---
name: clusterB
`,
			expected: []string{
				`name: clusterA`,
				`name: clusterB`,
			},
		},
	}
	for name, testSet := range testSets {
		t.Run(name, func(t *testing.T) {
			actual, err := SplitYAML([]byte(testSet.document))
			assert.NilError(t, err)
			assert.Equal(t, len(testSet.expected), len(actual))
			for idx := range testSet.expected {
				assert.Equal(t, testSet.expected[idx], strings.TrimSpace(string(actual[idx])))
			}
		})
	}
}

func TestYAMLEncoder(t *testing.T) {
	testSets := map[string]struct {
		values   []interface{}
		expected string
	}{
		"single value": {
			values: []interface{}{
				DummyContext{Name: "clusterA"},
			},
			expected: `Name: clusterA
`,
		},
		"single value with json tag": {
			values: []interface{}{
				DummyContextWithTag{Name: "clusterA"},
			},
			expected: `newName: clusterA
`,
		},
		"multiple values": {
			values: []interface{}{
				DummyContext{Name: "clusterA"},
				DummyContextWithTag{Name: "clusterB"},
			},
			expected: `Name: clusterA
---
newName: clusterB
`,
		},
	}
	for name, testSet := range testSets {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewYAMLEncoder(&buf)
			for _, v := range testSet.values {
				assert.NilError(t, enc.Encode(v))
			}
			assert.NilError(t, enc.Close())
			assert.Equal(t, testSet.expected, buf.String())
		})
	}
}
