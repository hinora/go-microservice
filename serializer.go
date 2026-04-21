package goservice

import (
	"encoding/json"

	"github.com/vmihailenco/msgpack/v5"
)

// SerializerType selects the wire serialization format for a broker.
type SerializerType int

const (
	// SerializerJSON is the default JSON serializer.
	SerializerJSON SerializerType = iota
	// SerializerMsgPack uses the compact MessagePack binary format.
	// All nodes in a cluster must use the same serializer.
	SerializerMsgPack
)

func SerializerJson(data interface{}) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func DeSerializerJson(jsonString string) (interface{}, error) {
	var data interface{}
	err := json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// SerializerMsgPackEncode encodes data to MessagePack bytes.
func SerializerMsgPackEncode(data interface{}) ([]byte, error) {
	return msgpack.Marshal(data)
}

// DeserializerMsgPackDecode decodes MessagePack bytes into a generic interface.
func DeserializerMsgPackDecode(b []byte) (interface{}, error) {
	var data interface{}
	err := msgpack.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
