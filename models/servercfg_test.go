package models

import (
	"encoding/json"
	"testing"
)

// Rule fields the builder doesn't model (e.g. `protocol`) must survive a
// read/write cycle. Dropping them leaves a rule with no match fields,
// which newer Xray versions refuse to start with.
func TestRoutingRulesRoundTrip(t *testing.T) {
	rules := []string{
		`{"type":"field","protocol":["bittorrent"],"outboundTag":"block"}`,
		`{"type":"field","network":"udp","port":"443","outboundTag":"block"}`,
		`{"type":"field","domain":["geosite:category-ads-all"],"outboundTag":"block"}`,
	}
	input := `{"routing":{"domainStrategy":"IPIfNonMatch","rules":[` + rules[0] + `,` + rules[1] + `,` + rules[2] + `]}}`

	cfg := ServerConfig{}
	if err := json.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var got struct {
		Routing struct {
			Rules []map[string]any `json:"rules"`
		} `json:"routing"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}

	if len(got.Routing.Rules) != len(rules) {
		t.Fatalf("expected %d rules, got %d", len(rules), len(got.Routing.Rules))
	}
	for i, raw := range rules {
		var want map[string]any
		if err := json.Unmarshal([]byte(raw), &want); err != nil {
			t.Fatal(err)
		}
		wantJson, _ := json.Marshal(want)
		gotJson, _ := json.Marshal(got.Routing.Rules[i])
		if string(wantJson) != string(gotJson) {
			t.Errorf("rule %d changed:\nwant %s\ngot  %s", i, wantJson, gotJson)
		}
	}
}
