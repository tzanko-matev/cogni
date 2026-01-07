package agent

import (
	"strings"
	"testing"
)

func TestAppendEnvironmentDiff(t *testing.T) {
	session := &Session{
		Ctx: TurnContext{
			CWD:            "/repo",
			ApprovalPolicy: "on-request",
			SandboxPolicy: SandboxPolicy{
				Mode:          "workspace-write",
				NetworkAccess: "enabled",
				WritableRoots: []string{"/tmp"},
				Shell:         "bash",
			},
		},
	}
	next := session.Ctx
	next.CWD = "/repo/sub"
	next.SandboxPolicy.NetworkAccess = "restricted"

	changed := AppendEnvironmentDiff(session, next)
	if !changed {
		t.Fatalf("expected diff")
	}
	if len(session.History) != 1 {
		t.Fatalf("expected history item")
	}
	content, ok := session.History[0].Content.(HistoryText)
	if !ok {
		t.Fatalf("expected text content")
	}
	if !strings.Contains(content.Text, "<cwd>/repo/sub</cwd>") {
		t.Fatalf("expected cwd change, got %q", content.Text)
	}
	if !strings.Contains(content.Text, "<network_access>restricted</network_access>") {
		t.Fatalf("expected network change, got %q", content.Text)
	}
	if strings.Contains(content.Text, "<sandbox_mode>") {
		t.Fatalf("did not expect sandbox mode")
	}
	if session.Ctx.CWD != "/repo/sub" {
		t.Fatalf("expected context update")
	}
}

func TestAppendEnvironmentDiffNoChanges(t *testing.T) {
	ctx := TurnContext{
		CWD:            "/repo",
		ApprovalPolicy: "on-request",
		SandboxPolicy: SandboxPolicy{
			Mode:          "workspace-write",
			NetworkAccess: "enabled",
			WritableRoots: []string{"/tmp"},
			Shell:         "bash",
		},
	}
	session := &Session{Ctx: ctx}
	if AppendEnvironmentDiff(session, ctx) {
		t.Fatalf("expected no diff")
	}
}
