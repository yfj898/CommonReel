package workflow

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// A recommendation cannot silently relax hard inputs. The model still plans
// tools; Go checks that changed hard fields have explicit evidence in the turn.
func (a *App) guardAgentWrite(s Session, name string, args map[string]any, message string) error {
	lower := strings.ToLower(message)
	if name == "set_policy" {
		b, _ := json.Marshal(args["policy"])
		var p Policy
		if json.Unmarshal(b, &p) != nil {
			return fail("INVALID_ARGUMENT", 400, "invalid policy")
		}
		if p.MaxRuntime != s.Policy.MaxRuntime {
			number := regexp.MustCompile(`(^|[^0-9])` + strconv.Itoa(p.MaxRuntime) + `([^0-9]|$)`)
			if !number.MatchString(lower) || !regexp.MustCompile(`runtime|minutes|min\b|under|at most|片长|分钟|不超过|以内`).MatchString(lower) {
				return fail("FORBIDDEN", 403, "a runtime change needs an explicit requested value")
			}
		}
		for _, id := range p.ProviderIDs {
			if !contains(s.Policy.ProviderIDs, id) && !strings.Contains(lower, strconv.Itoa(id)) && !strings.Contains(lower, strings.ToLower(providerName(id))) {
				return fail("FORBIDDEN", 403, "an added platform must be named in the request")
			}
		}
		if p.Rewatch != s.Policy.Rewatch {
			if !regexp.MustCompile(`rewatch|re-watch|watch again|重看|看过`).MatchString(lower) {
				return fail("FORBIDDEN", 403, "rewatch change must be explicit")
			}
			negative := regexp.MustCompile(`\b(no|not|never|false|cannot|can't)\b|不允许|不能|不要|禁止`).MatchString(lower)
			if p.Rewatch && negative {
				return fail("FORBIDDEN", 403, "cannot enable rewatch against an explicit negative request")
			}
		}
	}
	if name == "set_member" {
		mid, _ := args["member_id"].(string)
		i := memberIndex(s, mid)
		if i < 0 {
			return fail("INVALID_ARGUMENT", 400, "explicit current member required")
		}
		b, _ := json.Marshal(args["patch"])
		var patch PreferencePatch
		if json.Unmarshal(b, &patch) != nil {
			return fail("INVALID_ARGUMENT", 400, "invalid member patch")
		}
		if patch.Vetoes != nil {
			for _, old := range s.Members[i].Preferences.Vetoes {
				if !contains(*patch.Vetoes, old) && !regexp.MustCompile(`\b(remove|clear|replace|set)\b|去掉|取消|清空|改为|替换|不再否决`).MatchString(lower) {
					return fail("FORBIDDEN", 403, "removing an existing veto needs an explicit request")
				}
			}
		}
		if (patch.Seen != nil || len(patch.SeenAdd) > 0) && !regexp.MustCompile(`watched|seen|看过|看了|已看|删除历史|清空历史`).MatchString(lower) {
			return fail("FORBIDDEN", 403, "watch history changes must be explicit")
		}
	}
	return nil
}
func providerName(id int) string {
	if s := map[int]string{203: "Netflix", 26: "Amazon Prime", 372: "Disney+", 157: "Hulu", 387: "HBO Max", 388: "Peacock", 444: "Paramount+"}[id]; s != "" {
		return s
	}
	return "provider " + strconv.Itoa(id)
}
