package apimodels

import (
	"strings"
	"testing"
)

func TestSysinfoResponse_Decode_EmptyArrayLinkInfo(t *testing.T) {
	jsonStr := `{
		"api_version": "1.7",
		"link_info": []
	}`

	var resp SysinfoResponse
	err := resp.Decode(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if resp.SysinfoResponse1Point7.LinkInfo != nil {
		if len(resp.SysinfoResponse1Point7.LinkInfo) != 0 {
			t.Errorf("Expected empty LinkInfo, got %v", resp.SysinfoResponse1Point7.LinkInfo)
		}
	}
}

func TestSysinfoResponse_Decode_MapLinkInfo(t *testing.T) {
	jsonStr := `{
		"api_version": "1.7",
		"link_info": {
			"node1": {
				"hostname": "node1",
				"linkType": "RF",
				"olsrInterface": "wlan0",
				"linkQuality": 1.0,
				"neighborLinkQuality": 1.0
			}
		}
	}`

	var resp SysinfoResponse
	err := resp.Decode(strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(resp.SysinfoResponse1Point7.LinkInfo) != 1 {
		t.Errorf("Expected 1 LinkInfo entry, got %d", len(resp.SysinfoResponse1Point7.LinkInfo))
	}
}
