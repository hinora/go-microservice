package goservice

import (
	"testing"

	"github.com/mitchellh/mapstructure"
)

func TestSerializeDeserializeJSONDefault(t *testing.T) {
	b := &Broker{}

	payload, err := b.serializeWire(RequestTranferData{
		Params:       map[string]interface{}{"name": "Ada"},
		RequestId:    "req-1",
		ResponseId:   "res-1",
		CallingLevel: 2,
	})
	if err != nil {
		t.Fatalf("serialize JSON: %v", err)
	}

	decoded, err := b.deserializeWire(payload)
	if err != nil {
		t.Fatalf("deserialize JSON: %v", err)
	}

	var data RequestTranferData
	if err := mapstructure.Decode(decoded, &data); err != nil {
		t.Fatalf("decode JSON payload: %v", err)
	}
	if data.RequestId != "req-1" || data.ResponseId != "res-1" || data.CallingLevel != 2 {
		t.Fatalf("unexpected decoded JSON payload: %#v", data)
	}
}

func TestSerializeDeserializeMsgPack(t *testing.T) {
	b := &Broker{Config: BrokerConfig{Serializer: SerializerMsgPack}}

	payload, err := b.serializeWire(ResponseTranferData{
		Data:           map[string]interface{}{"ok": true},
		ResponseId:     "res-1",
		ResponseNodeId: "node-1",
		ResponseTime:   123,
	})
	if err != nil {
		t.Fatalf("serialize MessagePack: %v", err)
	}

	decoded, err := b.deserializeWire(payload)
	if err != nil {
		t.Fatalf("deserialize MessagePack: %v", err)
	}

	var data ResponseTranferData
	if err := mapstructure.Decode(decoded, &data); err != nil {
		t.Fatalf("decode MessagePack payload: %v", err)
	}
	if data.ResponseId != "res-1" || data.ResponseNodeId != "node-1" || data.ResponseTime != 123 {
		t.Fatalf("unexpected decoded MessagePack payload: %#v", data)
	}
}
