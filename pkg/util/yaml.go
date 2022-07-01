package util

import (
	"bytes"
	"io"

	goyaml "gopkg.in/yaml.v2"
	"sigs.k8s.io/yaml"
)

func SplitYAML(resources []byte) ([][]byte, error) {
	dec := goyaml.NewDecoder(bytes.NewReader(resources))

	var res [][]byte
	for {
		var value interface{}
		err := dec.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		valueBytes, err := goyaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		res = append(res, valueBytes)
	}
	return res, nil
}

type YAMLEncoder struct {
	encoder *goyaml.Encoder
}

func NewYAMLEncoder(w io.Writer) *YAMLEncoder {
	return &YAMLEncoder{
		encoder: goyaml.NewEncoder(w),
	}
}

func (e *YAMLEncoder) Encode(v interface{}) (err error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	var doc interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	return e.encoder.Encode(doc)
}

func (e *YAMLEncoder) Close() (err error) {
	return e.encoder.Close()
}
