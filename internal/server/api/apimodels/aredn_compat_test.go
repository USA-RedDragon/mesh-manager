package apimodels

import (
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
