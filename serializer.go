package goservice

import (
	"bytes"
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

// Serialize encodes data using the selected wire serializer.
func Serialize(data interface{}, serializer SerializerType) ([]byte, error) {
	switch serializer {
	case SerializerMsgPack:
		return SerializerMsgPackEncode(data)
	case SerializerJSON:
		fallthrough
	default:
		return json.Marshal(data)
	}
}

// Deserialize decodes wire data using the selected serializer.
func Deserialize(data []byte, serializer SerializerType) (interface{}, error) {
	switch serializer {
	case SerializerMsgPack:
		return DeserializerMsgPackDecode(data)
	case SerializerJSON:
		fallthrough
	default:
		var decoded interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	}
}

func (b *Broker) serializeWire(data interface{}) ([]byte, error) {
	return Serialize(data, b.Config.Serializer)
}

func (b *Broker) deserializeWire(data []byte) (interface{}, error) {
	return Deserialize(data, b.Config.Serializer)
}

// SerializerMsgPackEncode encodes data to MessagePack bytes.
func SerializerMsgPackEncode(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.SetCustomStructTag("json")
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DeserializerMsgPackDecode decodes MessagePack bytes into a generic interface.
func DeserializerMsgPackDecode(b []byte) (interface{}, error) {
	var data interface{}
	dec := msgpack.NewDecoder(bytes.NewReader(b))
	dec.SetCustomStructTag("json")
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}
