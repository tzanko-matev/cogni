package agent

import (
	"fmt"
	"strings"
)

func AppendEnvironmentDiff(session *Session, next TurnContext) bool {
	if session == nil {
		return false
	}
	diff := formatEnvironmentDiff(session.Ctx, next)
	if diff == "" {
		return false
	}
	session.History = append(session.History, HistoryItem{
		Role:    "user",
		Content: diff,
	})
	session.Ctx = next
	return true
}

func formatEnvironmentDiff(current, next TurnContext) string {
	var builder strings.Builder
	changed := false
	builder.WriteString("<environment_diff>\n")

	if current.CWD != next.CWD {
		changed = true
		builder.WriteString(fmt.Sprintf("  <cwd>%s</cwd>\n", next.CWD))
	}
	if current.ApprovalPolicy != next.ApprovalPolicy {
		changed = true
		builder.WriteString(fmt.Sprintf("  <approval_policy>%s</approval_policy>\n", next.ApprovalPolicy))
	}
	if current.SandboxPolicy.Mode != next.SandboxPolicy.Mode {
		changed = true
		builder.WriteString(fmt.Sprintf("  <sandbox_mode>%s</sandbox_mode>\n", next.SandboxPolicy.Mode))
	}
	if current.SandboxPolicy.NetworkAccess != next.SandboxPolicy.NetworkAccess {
		changed = true
		builder.WriteString(fmt.Sprintf("  <network_access>%s</network_access>\n", next.SandboxPolicy.NetworkAccess))
	}
	if !stringSliceEqual(current.SandboxPolicy.WritableRoots, next.SandboxPolicy.WritableRoots) {
		changed = true
		builder.WriteString("  <writable_roots>\n")
		for _, root := range next.SandboxPolicy.WritableRoots {
			builder.WriteString(fmt.Sprintf("    <root>%s</root>\n", root))
		}
		builder.WriteString("  </writable_roots>\n")
	}
	if current.SandboxPolicy.Shell != next.SandboxPolicy.Shell {
		changed = true
		builder.WriteString(fmt.Sprintf("  <shell>%s</shell>\n", next.SandboxPolicy.Shell))
	}

	if !changed {
		return ""
	}
	builder.WriteString("</environment_diff>")
	return builder.String()
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
