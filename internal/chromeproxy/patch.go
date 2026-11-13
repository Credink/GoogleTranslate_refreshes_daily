package chromeproxy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PatchProfileJSON rewrites one Omega profile value (chrome.storage JSON) so
// managed translate domains use DIRECT / bypass. Returns changed=false when
// already correct or the value is not a recognized profile object.
func PatchProfileJSON(raw []byte, domains []string) (out []byte, changed bool, detail string, err error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, false, "", err
	}
	pt := jsonString(obj["profileType"])
	switch pt {
	case "FixedProfile":
		return patchFixedProfile(obj, raw, domains)
	case "SwitchProfile":
		return patchSwitchProfile(obj, raw, domains)
	case "RuleListProfile":
		return patchRuleListProfile(obj, raw, domains)
	default:
		return raw, false, "skip unsupported profileType=" + pt, nil
	}
}

func patchFixedProfile(obj map[string]json.RawMessage, raw []byte, domains []string) ([]byte, bool, string, error) {
	var list []map[string]any
	if b, ok := obj["bypassList"]; ok && len(b) > 0 && string(b) != "null" {
		if err := json.Unmarshal(b, &list); err != nil {
			return nil, false, "", fmt.Errorf("bypassList: %w", err)
		}
	}
	list, typoFixed := rewriteBypassTypos(list)
	needed := make([]string, 0, len(domains))
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if bypassHasDomain(list, d) {
			continue
		}
		needed = append(needed, d)
	}
	if len(needed) == 0 && !typoFixed {
		return raw, false, "FixedProfile bypass already OK", nil
	}
	for _, d := range needed {
		list = append(list, map[string]any{
			"conditionType": "BypassCondition",
			"pattern":       d,
		})
	}
	obj["bypassList"] = mustJSON(list)
	bumpRevision(obj)
	out, err := json.Marshal(obj)
	if err != nil {
		return nil, false, "", err
	}
	detail := fmt.Sprintf("FixedProfile +%d bypass", len(needed))
	if typoFixed {
		detail += " (fixed typos)"
	}
	return out, true, detail, nil
}

func patchSwitchProfile(obj map[string]json.RawMessage, raw []byte, domains []string) ([]byte, bool, string, error) {
	var rules []map[string]any
	if b, ok := obj["rules"]; ok && len(b) > 0 && string(b) != "null" {
		if err := json.Unmarshal(b, &rules); err != nil {
			return nil, false, "", fmt.Errorf("rules: %w", err)
		}
	}
	rules, typoFixed := rewriteSwitchTypos(rules)
	rules, forced := forceSwitchDirect(rules, domains)
	rules, added := prependSwitchDirect(rules, domains)
	if !typoFixed && !forced && added == 0 {
		return raw, false, "SwitchProfile rules already OK", nil
	}
	obj["rules"] = mustJSON(rules)
	bumpRevision(obj)
	out, err := json.Marshal(obj)
	if err != nil {
		return nil, false, "", err
	}
	detail := fmt.Sprintf("SwitchProfile +%d direct rules", added)
	if forced {
		detail += ", retargeted existing"
	}
	if typoFixed {
		detail += " (fixed typos)"
	}
	return out, true, detail, nil
}

func patchRuleListProfile(obj map[string]json.RawMessage, raw []byte, domains []string) ([]byte, bool, string, error) {
	list := jsonString(obj["ruleList"])
	if list == "" {
		return raw, false, "RuleListProfile empty ruleList", nil
	}
	newList, changed := patchSwitchyRuleList(list, domains)
	if !changed {
		return raw, false, "RuleListProfile already OK", nil
	}
	obj["ruleList"] = mustJSON(newList)
	// Stale compiled PAC would disagree with ruleList; clear so Omega rebuilds.
	if _, ok := obj["pacScript"]; ok {
		obj["pacScript"] = mustJSON("")
	}
	bumpRevision(obj)
	out, err := json.Marshal(obj)
	if err != nil {
		return nil, false, "", err
	}
	return out, true, "RuleListProfile updated ruleList", nil
}

func bumpRevision(obj map[string]json.RawMessage) {
	obj["revision"] = mustJSON(fmt.Sprintf("%x", time.Now().UnixMilli()))
}

func rewriteBypassTypos(list []map[string]any) ([]map[string]any, bool) {
	changed := false
	for _, item := range list {
		p, _ := item["pattern"].(string)
		fixed := FixTranslateTypo(p)
		if fixed != p {
			item["pattern"] = fixed
			changed = true
		}
	}
	return list, changed
}

func rewriteSwitchTypos(rules []map[string]any) ([]map[string]any, bool) {
	changed := false
	for _, rule := range rules {
		cond, _ := rule["condition"].(map[string]any)
		if cond == nil {
			continue
		}
		p, _ := cond["pattern"].(string)
		fixed := FixTranslateTypo(p)
		if fixed != p {
			cond["pattern"] = fixed
			changed = true
		}
	}
	return rules, changed
}

func forceSwitchDirect(rules []map[string]any, domains []string) ([]map[string]any, bool) {
	changed := false
	want := domainSet(domains)
	for _, rule := range rules {
		cond, _ := rule["condition"].(map[string]any)
		if cond == nil {
			continue
		}
		p, _ := cond["pattern"].(string)
		if !patternTargetsManaged(p, want) {
			continue
		}
		if name, _ := rule["profileName"].(string); !strings.EqualFold(name, "direct") {
			rule["profileName"] = "direct"
			changed = true
		}
	}
	return rules, changed
}

func prependSwitchDirect(rules []map[string]any, domains []string) ([]map[string]any, int) {
	existing := map[string]bool{}
	for _, rule := range rules {
		cond, _ := rule["condition"].(map[string]any)
		if cond == nil {
			continue
		}
		p, _ := cond["pattern"].(string)
		existing[strings.ToLower(strings.TrimSpace(p))] = true
	}
	var add []map[string]any
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		key := strings.ToLower(d)
		if existing[key] {
			continue
		}
		existing[key] = true
		add = append(add, map[string]any{
			"condition": map[string]any{
				"conditionType": "HostWildcardCondition",
				"pattern":       d,
			},
			"profileName": "direct",
		})
	}
	if len(add) == 0 {
		return rules, 0
	}
	return append(add, rules...), len(add)
}

func bypassHasDomain(list []map[string]any, domain string) bool {
	want := strings.ToLower(domain)
	for _, item := range list {
		p, _ := item["pattern"].(string)
		p = strings.ToLower(strings.TrimSpace(p))
		if p == want || p == "*"+want || p == "*"+want+"*" {
			return true
		}
		// Already covered by a broad managed-host pattern we would add exactly.
		if p == want {
			return true
		}
	}
	return false
}

func patternTargetsManaged(pattern string, want map[string]struct{}) bool {
	p := strings.ToLower(strings.TrimSpace(pattern))
	p = strings.TrimPrefix(p, "*.")
	p = strings.Trim(p, "*")
	if _, ok := want[p]; ok {
		return true
	}
	for d := range want {
		if p == d {
			return true
		}
	}
	return false
}

func domainSet(domains []string) map[string]struct{} {
	m := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			m[d] = struct{}{}
		}
	}
	return m
}

// FixTranslateTypo corrects common host typos like "ranslate.googleapis.com".
func FixTranslateTypo(pattern string) string {
	if pattern == "" {
		return pattern
	}
	if strings.Contains(strings.ToLower(pattern), "translate") {
		return pattern
	}
	// Only fix when the typo fragment is present.
	replacer := strings.NewReplacer(
		"ranslate", "translate",
		"Ranslate", "Translate",
		"RANSLATE", "TRANSLATE",
	)
	return replacer.Replace(pattern)
}

func patchSwitchyRuleList(text string, domains []string) (string, bool) {
	lines := strings.Split(text, "\n")
	changed := false
	have := map[string]bool{}
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		fixed := FixTranslateTypo(trim)
		if fixed != trim {
			lines[i] = fixed
			trim = fixed
			changed = true
		}
		// Track exclusive / normal host lines.
		key := strings.TrimLeft(trim, "!")
		key = strings.ToLower(strings.TrimSpace(key))
		if key != "" && !strings.HasPrefix(key, "[") && !strings.HasPrefix(key, ";") && !strings.HasPrefix(key, "@") {
			have[key] = true
			have[strings.TrimPrefix(key, "*.")] = true
		}
		// If a line explicitly routes managed domains via "+proxy", force direct.
		if strings.Contains(trim, " +") && !strings.HasPrefix(trim, ";") {
			for d := range domainSet(domains) {
				if strings.Contains(strings.ToLower(trim), d) && !strings.HasSuffix(strings.ToLower(trim), " +direct") {
					// best-effort: exclusive direct form
					lines[i] = "!" + d
					changed = true
					have[d] = true
				}
			}
		}
	}
	var add []string
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		key := strings.ToLower(d)
		if have[key] || have["*"+key] || have["*."+key] {
			continue
		}
		// Exclusive rule → DIRECT in Switchy rule lists when match profile is proxy.
		add = append(add, "!"+d)
		have[key] = true
	}
	if len(add) == 0 && !changed {
		return text, false
	}
	if len(add) > 0 {
		// Insert after header block.
		insertAt := 0
		for i, line := range lines {
			trim := strings.TrimSpace(line)
			if trim == "" || strings.HasPrefix(trim, "[") || strings.HasPrefix(trim, ";") || strings.HasPrefix(trim, "@") {
				insertAt = i + 1
				continue
			}
			break
		}
		tail := append([]string{}, lines[insertAt:]...)
		lines = append(lines[:insertAt], append(add, tail...)...)
		changed = true
	}
	return strings.Join(lines, "\n"), changed
}

func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return strings.Trim(string(raw), `"`)
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return b
}
