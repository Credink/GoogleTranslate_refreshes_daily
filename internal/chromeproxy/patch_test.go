package chromeproxy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFixTranslateTypo(t *testing.T) {
	cases := map[string]string{
		"ranslate.googleapis.com":  "translate.googleapis.com",
		"*ranslate.google.com":     "*translate.google.com",
		"translate.googleapis.com": "translate.googleapis.com",
		"*baidu.*":                 "*baidu.*",
	}
	for in, want := range cases {
		if got := FixTranslateTypo(in); got != want {
			t.Fatalf("FixTranslateTypo(%q)=%q want %q", in, got, want)
		}
	}
}

func TestPatchFixedProfileAddsBypass(t *testing.T) {
	raw := []byte(`{"bypassList":[{"conditionType":"BypassCondition","pattern":"localhost"}],"name":"proxy","profileType":"FixedProfile","revision":"abc"}`)
	out, changed, detail, err := PatchProfileJSON(raw, []string{"translate.googleapis.com", "translate.google.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	if !strings.Contains(detail, "FixedProfile") {
		t.Fatalf("detail=%s", detail)
	}
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatal(err)
	}
	list, _ := obj["bypassList"].([]any)
	if len(list) < 3 {
		t.Fatalf("bypassList len=%d", len(list))
	}
	joined, _ := json.Marshal(list)
	if !strings.Contains(string(joined), "translate.googleapis.com") {
		t.Fatalf("missing domain: %s", joined)
	}
}

func TestPatchFixedProfileIdempotent(t *testing.T) {
	raw := []byte(`{"bypassList":[{"conditionType":"BypassCondition","pattern":"translate.googleapis.com"}],"name":"proxy","profileType":"FixedProfile"}`)
	_, changed, _, err := PatchProfileJSON(raw, []string{"translate.googleapis.com"})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no change")
	}
}

func TestPatchSwitchPrependsDirect(t *testing.T) {
	raw := []byte(`{
	  "name":"auto switch","profileType":"SwitchProfile","defaultProfileName":"direct",
	  "rules":[{"condition":{"conditionType":"HostWildcardCondition","pattern":"*googleapis.com"},"profileName":"proxy"}]
	}`)
	out, changed, _, err := PatchProfileJSON(raw, []string{"translate.googleapis.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	var obj struct {
		Rules []struct {
			ProfileName string `json:"profileName"`
			Condition   struct {
				Pattern string `json:"pattern"`
			} `json:"condition"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatal(err)
	}
	if len(obj.Rules) < 2 {
		t.Fatalf("rules=%d", len(obj.Rules))
	}
	if obj.Rules[0].Condition.Pattern != "translate.googleapis.com" || obj.Rules[0].ProfileName != "direct" {
		t.Fatalf("first rule = %+v", obj.Rules[0])
	}
}

func TestPatchSwitchRetargetsExisting(t *testing.T) {
	raw := []byte(`{
	  "name":"auto switch","profileType":"SwitchProfile",
	  "rules":[{"condition":{"conditionType":"HostWildcardCondition","pattern":"translate.googleapis.com"},"profileName":"proxy"}]
	}`)
	out, changed, _, err := PatchProfileJSON(raw, []string{"translate.googleapis.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected retarget")
	}
	if !strings.Contains(string(out), `"profileName":"direct"`) {
		t.Fatalf("out=%s", out)
	}
}

func TestPatchFixesTypoInBypass(t *testing.T) {
	raw := []byte(`{"bypassList":[{"conditionType":"BypassCondition","pattern":"ranslate.googleapis.com"}],"profileType":"FixedProfile","name":"proxy"}`)
	out, changed, detail, err := PatchProfileJSON(raw, []string{"translate.googleapis.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected typo fix")
	}
	if !strings.Contains(detail, "typo") {
		t.Fatalf("detail=%s", detail)
	}
	if strings.Contains(string(out), `"pattern":"ranslate.googleapis.com"`) {
		t.Fatalf("typo remained: %s", out)
	}
	if !strings.Contains(string(out), `"pattern":"translate.googleapis.com"`) {
		t.Fatalf("expected fixed domain: %s", out)
	}
}

func TestPatchSwitchyRuleList(t *testing.T) {
	in := "[SwitchyOmega Conditions]\n; note\n*github.*\n"
	out, changed := patchSwitchyRuleList(in, []string{"translate.googleapis.com"})
	if !changed {
		t.Fatal("expected change")
	}
	if !strings.Contains(out, "!translate.googleapis.com") {
		t.Fatalf("out=%s", out)
	}
}

func TestPatchIdempotentSecondPass(t *testing.T) {
	raw := []byte(`{"bypassList":[{"conditionType":"BypassCondition","pattern":"localhost"}],"name":"proxy","profileType":"FixedProfile","revision":"abc"}`)
	domains := []string{"translate.googleapis.com", "translate.google.com"}
	out1, changed1, _, err := PatchProfileJSON(raw, domains)
	if err != nil || !changed1 {
		t.Fatalf("first pass: changed=%v err=%v", changed1, err)
	}
	out2, changed2, detail, err := PatchProfileJSON(out1, domains)
	if err != nil {
		t.Fatal(err)
	}
	if changed2 {
		t.Fatalf("second pass must not rewrite, detail=%s out=%s", detail, out2)
	}
}

func TestConfigureBypassOptsSkipsWhenDomainsUnchanged(t *testing.T) {
	domains := []string{"translate.googleapis.com"}
	key := DomainsKey(domains)
	rep := ConfigureBypassOpts(domains, Options{
		Force:            false,
		PriorOK:          true,
		SkipIfDomainsKey: key,
	})
	if !rep.SkippedAuto || rep.Status != StatusOK {
		t.Fatalf("expected auto-skip, got %+v", rep)
	}
	if rep.Changed != 0 {
		t.Fatalf("auto-skip must not change anything: %+v", rep)
	}
}
