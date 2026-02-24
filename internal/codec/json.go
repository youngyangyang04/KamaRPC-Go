package codec

import "encoding/json"

const JSON Type = 1

type jSONCodec struct{}

func (j *jSONCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (j *jSONCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func init() {
	Register(JSON, func() Codec {
		return &jSONCodec{}
	})
}
