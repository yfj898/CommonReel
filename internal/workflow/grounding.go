package workflow

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func writeReply(steps []AgentStep, zh bool) string {
	var b strings.Builder
	for _, step := range steps {
		if !step.OK || step.Tool != "set_member" && step.Tool != "set_policy" {
			continue
		}
		target := "shared room limits"
		if step.Tool == "set_member" {
			target = "member " + step.Target + " tonight inputs"
		}
		fields := append([]string{}, step.Fields...)
		sort.Strings(fields)
		if zh {
			fmt.Fprintf(&b, "已处理%s，提交房间版本 %d，结果 %s。字段：%s。\n", target, step.Revision, step.Code, strings.Join(fields, ", "))
		} else {
			fmt.Fprintf(&b, "Processed %s at room revision %d: %s. Fields: %s.\n", target, step.Revision, step.Code, strings.Join(fields, ", "))
		}
	}
	if zh {
		b.WriteString("请查看页面当前值；其他成员随后可能继续修改。没有执行表态、保存或删除。")
	} else {
		b.WriteString("See the page for current values; other members may make later changes. No response, save or deletion was performed.")
	}
	return b.String()
}

// The model plans tools; this renderer owns factual claims displayed to people.
// It never decodes provider IDs into guessed names or implies account entitlement.
func groundedReply(list *Shortlist, explanation *Candidate, p Policy, zh bool) string {
	var b strings.Builder
	if list != nil {
		if zh {
			fmt.Fprintf(&b, "在 %d 部目录电影中，本轮视图核实供应 %d 部，因供应未知排除 %d 部。\n", list.CatalogTotal, list.SupplyVerified, list.Unknown)
		} else {
			fmt.Fprintf(&b, "Catalog: %d movies. Supply verified in this view: %d. Excluded for unknown supply: %d.\n", list.CatalogTotal, list.SupplyVerified, list.Unknown)
		}
		if len(list.Candidates) == 0 {
			if zh {
				b.WriteString("没有满足当前条件的候选。请明确调整条件后重试；系统没有放宽条件。\n")
			} else {
				b.WriteString("No candidates meet the current inputs. Change limits explicitly or retry unknown supply; nothing was relaxed.\n")
			}
		}
		for _, c := range list.Candidates {
			writeCandidate(&b, c, p, false, zh)
		}
	}
	if explanation != nil {
		if zh {
			b.WriteString("\n已核实候选解释\n")
		} else {
			b.WriteString("\nVerified candidate explanation\n")
		}
		writeCandidate(&b, *explanation, p, true, zh)
	}
	if zh {
		b.WriteString("\n分数是估计，不代表同意或满意。订阅供应不保证个人账户权益；表态和保存请使用页面按钮。")
	} else {
		b.WriteString("\nScores are estimates, not acceptance or satisfaction. Subscription listings do not verify personal account entitlement. Record responses and save with page buttons.")
	}
	return b.String()
}
func writeCandidate(b *strings.Builder, c Candidate, p Policy, detail, zh bool) {
	names := []string{}
	for _, o := range c.Availability.Offers {
		if o.Region == "US" && o.Monetization == "sub" && contains(p.ProviderIDs, o.ProviderID) {
			name := o.Name
			if name == "" {
				name = fmt.Sprintf("provider %d", o.ProviderID)
			}
			if !contains(names, name) {
				names = append(names, name)
			}
		}
	}
	if zh {
		fmt.Fprintf(b, "• %s（%d），%d min；所选美国订阅供应：%s；小组估计 %.2f/5。\n", c.Film.Title, c.Film.Year, c.Film.Runtime, strings.Join(names, ", "), c.Score)
	} else {
		fmt.Fprintf(b, "• %s (%d), %d min. Selected US subscriptions: %s. Estimated group score %.2f/5.\n", c.Film.Title, c.Film.Year, c.Film.Runtime, strings.Join(names, ", "), c.Score)
	}
	if detail {
		if zh {
			fmt.Fprintf(b, "  最低个人估计 %.2f，平均估计 %.2f；默认分数为 0.65 × 最低 + 0.35 × 平均。供应查询于 %s。\n", c.Minimum, c.Mean, c.Availability.FetchedAt.UTC().Format(time.RFC3339))
		} else {
			fmt.Fprintf(b, "  Minimum member estimate %.2f; mean %.2f. Default score = 0.65 × minimum + 0.35 × mean. Supply queried %s.\n", c.Minimum, c.Mean, c.Availability.FetchedAt.UTC().Format(time.RFC3339))
		}
	}
}
