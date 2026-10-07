package api

import (
	"testing"

	"github.com/xtls/xray-core/app/stats/command"
)

func TestParseStatNameValid(t *testing.T) {
	name, link, statType, ok := parseStatName("user>>>alice@example.com>>>traffic>>>uplink")
	if !ok {
		t.Fatal("expected valid stat name")
	}
	if name != "alice@example.com" {
		t.Fatalf("unexpected name: got %q", name)
	}
	if link != "traffic" {
		t.Fatalf("unexpected link: got %q", link)
	}
	if statType != "uplink" {
		t.Fatalf("unexpected type: got %q", statType)
	}
}

func TestParseStatNameRejectsMalformed(t *testing.T) {
	tests := []string{
		"user>>>alice@example.com>>>online",  // too short
		"user>>>>>>traffic>>>uplink",         // empty name
		"user>>>alice@example.com>>>>>>uplink", // empty link
		"user>>>alice@example.com>>>traffic>>>", // empty type
	}

	for _, raw := range tests {
		if _, _, _, ok := parseStatName(raw); ok {
			t.Fatalf("expected malformed stat name to be rejected: %q", raw)
		}
	}
}

func TestBuildStatResponseSkipsMalformedAndMapsFields(t *testing.T) {
	resp := buildStatResponse([]*command.Stat{
		{Name: "user>>>alice@example.com>>>traffic>>>uplink", Value: 123},
		{Name: "user>>>alice@example.com>>>traffic>>>downlink", Value: 456},
		{Name: "user>>>alice@example.com>>>online", Value: 1},
	})

	if len(resp.GetStats()) != 2 {
		t.Fatalf("expected 2 valid stats, got %d", len(resp.GetStats()))
	}

	if resp.GetStats()[0].GetName() != "alice@example.com" || resp.GetStats()[0].GetLink() != "traffic" || resp.GetStats()[0].GetType() != "uplink" || resp.GetStats()[0].GetValue() != 123 {
		t.Fatalf("unexpected first stat mapping: %+v", resp.GetStats()[0])
	}
	if resp.GetStats()[1].GetName() != "alice@example.com" || resp.GetStats()[1].GetLink() != "traffic" || resp.GetStats()[1].GetType() != "downlink" || resp.GetStats()[1].GetValue() != 456 {
		t.Fatalf("unexpected second stat mapping: %+v", resp.GetStats()[1])
	}
}

func TestParseOnlineMetricNameValid(t *testing.T) {
	tests := []struct {
		raw   string
		email string
	}{
		{"user>>>alice@example.com>>>online", "alice@example.com"},
		{"user>>>67579385:de-mega-cdn>>>online", "67579385:de-mega-cdn"},
	}

	for _, tt := range tests {
		email, ok := parseOnlineMetricName(tt.raw)
		if !ok {
			t.Fatalf("expected %q to parse", tt.raw)
		}
		if email != tt.email {
			t.Fatalf("unexpected email: got %q, want %q", email, tt.email)
		}
	}
}

func TestParseOnlineMetricNameRejectsMalformed(t *testing.T) {
	tests := []string{
		"user>>>alice@example.com>>>traffic>>>uplink", // wrong suffix
		"alice@example.com>>>online",                  // missing prefix
		"user>>>online",                               // empty email
		"",
	}

	for _, raw := range tests {
		if _, ok := parseOnlineMetricName(raw); ok {
			t.Fatalf("expected malformed metric name to be rejected: %q", raw)
		}
	}
}

func TestGetUsersOnlineStatsStripsMetricNames(t *testing.T) {
	names := []string{
		"user>>>alice@example.com>>>online",
		"user>>>67579385:de-mega-cdn>>>online",
		"inbound>>>some-tag>>>traffic>>>uplink", // not a user-online metric, must be skipped
	}

	emails := make([]string, 0, len(names))
	for _, raw := range names {
		if email, ok := parseOnlineMetricName(raw); ok {
			emails = append(emails, email)
		}
	}

	if len(emails) != 2 {
		t.Fatalf("expected 2 emails, got %d: %+v", len(emails), emails)
	}
	if emails[0] != "alice@example.com" || emails[1] != "67579385:de-mega-cdn" {
		t.Fatalf("unexpected emails: %+v", emails)
	}
}
