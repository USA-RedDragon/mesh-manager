package apimodels

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSysinfoResponse_Decode_NilFreq(t *testing.T) {
	jsonStr := `{
		"api_version": "1.7",
		"meshrf": {
			"ssid": "AREDN-10-v3",
			"channel": "36",
			"status": "on",
			"mode": "adhoc",
			"chanbw": "10",
			"freq": "nil"
		}
	}`

	var resp SysinfoResponse
	err := resp.Decode(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	rf := resp.SysinfoResponse1Point7.MeshRF
	if rf.Frequency != 0 {
		t.Errorf("Expected Frequency to be 0, got %v", rf.Frequency)
	}
}

func TestSysinfoResponse_Decode_ValidFreq(t *testing.T) {
	jsonStr := `{
		"api_version": "1.7",
		"meshrf": {
			"ssid": "AREDN-10-v3",
			"channel": "36",
			"status": "on",
			"mode": "adhoc",
			"chanbw": "10",
			"freq": "5180"
		}
	}`

	var resp SysinfoResponse
	err := resp.Decode(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	rf := resp.SysinfoResponse1Point7.MeshRF
	if rf.Frequency != 5180 {
		t.Errorf("Expected Frequency to be 5180, got %v", rf.Frequency)
	}
}

func TestSysinfoResponse_Decode_1Point11_UserBlocks_Encode(t *testing.T) {
	jsonStr := `{
		"api_version": "1.11",
		"node": "TEST-NODE",
		"lqm": {
			"enabled": true,
			"config": {
				"min_snr": 15,
				"margin_snr": 1,
				"min_distance": 0,
				"max_distance": 80550,
				"auto_distance": 0,
				"min_quality": 50,
				"margin_quality": 1,
				"ping_penalty": 5,
				"user_blocks": {"10.0.0.1": true, "10.0.0.2": true},
				"user_allowlist": []
			},
			"info": {}
		}
	}`

	var resp SysinfoResponse
	err := resp.Decode(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	obj := resp.GetObject()
	_, err = json.Marshal(map[string]any{
		"data": obj,
	})
	if err != nil {
		t.Fatalf("Encode failed (this was the walker bug): %v", err)
	}
}
