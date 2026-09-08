package xray

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pasarguard/node/backend/xray/api"
	"github.com/pasarguard/node/common"
)

func TestScopedEmailFormat(t *testing.T) {
	got := scopedEmail("12345", "VLESS-REALITY-1")
	want := "12345:VLESS-REALITY-1"
	if got != want {
		t.Fatalf("scopedEmail: got %q want %q", got, want)
	}
}

func TestWithEmailReturnsCopyNotMutatingOriginal(t *testing.T) {
	original := &api.VlessAccount{
		BaseAccount: api.BaseAccount{Email: "12345", Level: 0},
		ID:          uuid.New(),
		Flow:        "xtls-rprx-vision",
	}

	scoped := withEmail(original, "12345:tag-a")

	if original.Email != "12345" {
		t.Fatalf("original account was mutated: got email %q", original.Email)
	}
	if scoped.GetEmail() != "12345:tag-a" {
		t.Fatalf("scoped account: got email %q want %q", scoped.GetEmail(), "12345:tag-a")
	}

	scopedVless, ok := scoped.(*api.VlessAccount)
	if !ok {
		t.Fatalf("unexpected type: %T", scoped)
	}
	if scopedVless.ID != original.ID || scopedVless.Flow != original.Flow {
		t.Fatalf("withEmail dropped fields: got %+v, from %+v", scopedVless, original)
	}
}

// The same *VlessAccount pointer (from ProxySettings, built once per user) is reused
// across every inbound that user is active on - withEmail must produce an independent
// copy each time, or the second inbound's scoped email would leak into the first.
func TestWithEmailIndependentAcrossRepeatedCalls(t *testing.T) {
	shared := &api.VlessAccount{
		BaseAccount: api.BaseAccount{Email: "12345", Level: 0},
		ID:          uuid.New(),
	}

	first := withEmail(shared, scopedEmail("12345", "inbound-a"))
	second := withEmail(shared, scopedEmail("12345", "inbound-b"))

	if first.GetEmail() != "12345:inbound-a" {
		t.Fatalf("first: got %q", first.GetEmail())
	}
	if second.GetEmail() != "12345:inbound-b" {
		t.Fatalf("second: got %q", second.GetEmail())
	}
	if first.GetEmail() == second.GetEmail() {
		t.Fatalf("first and second scoped emails collided: %q", first.GetEmail())
	}
}

func TestInboundSyncUsersKeysClientsByScopedEmailPerProtocol(t *testing.T) {
	makeUser := func(id string, proxy *common.Proxy) *common.User {
		return &common.User{
			Email:    id,
			Inbounds: []string{"tag-x"},
			Proxies:  proxy,
		}
	}

	cases := []struct {
		name     string
		protocol string
		user     *common.User
	}{
		{
			name:     "vless",
			protocol: Vless,
			user:     makeUser("111", &common.Proxy{Vless: &common.Vless{Id: uuid.New().String()}}),
		},
		{
			name:     "vmess",
			protocol: Vmess,
			user:     makeUser("222", &common.Proxy{Vmess: &common.Vmess{Id: uuid.New().String()}}),
		},
		{
			name:     "trojan",
			protocol: Trojan,
			user:     makeUser("333", &common.Proxy{Trojan: &common.Trojan{Password: "pw"}}),
		},
		{
			name:     "shadowsocks",
			protocol: Shadowsocks,
			user:     makeUser("444", &common.Proxy{Shadowsocks: &common.Shadowsocks{Password: "pw", Method: "aes-256-gcm"}}),
		},
		{
			name:     "hysteria",
			protocol: Hysteria,
			user:     makeUser("555", &common.Proxy{Hysteria: &common.Hysteria{Auth: "auth"}}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inbound := &Inbound{Tag: "tag-x", Protocol: tc.protocol}
			inbound.syncUsers([]*common.User{tc.user})

			wantKey := scopedEmail(tc.user.GetEmail(), "tag-x")
			account, ok := inbound.clients[wantKey]
			if !ok {
				t.Fatalf("client map has no entry for scoped key %q, keys: %v", wantKey, keysOf(inbound.clients))
			}
			if account.GetEmail() != wantKey {
				t.Fatalf("stored account email: got %q want %q", account.GetEmail(), wantKey)
			}
			if _, stillRaw := inbound.clients[tc.user.GetEmail()]; stillRaw {
				t.Fatalf("client map should not contain the unscoped email %q", tc.user.GetEmail())
			}
		})
	}
}

func TestBuildInboundUpdatesScopesAccountsAndRemovals(t *testing.T) {
	activeUser := &common.User{
		Email:    "666",
		Inbounds: []string{"tag-a"},
		Proxies:  &common.Proxy{Vless: &common.Vless{Id: uuid.New().String()}},
	}
	inactiveUser := &common.User{
		Email:    "777",
		Inbounds: []string{}, // not active on tag-a
		Proxies:  &common.Proxy{Vless: &common.Vless{Id: uuid.New().String()}},
	}

	cfg := &Config{
		InboundConfigs: []*Inbound{
			{Tag: "tag-a", Protocol: Vless},
		},
	}

	_, updates := cfg.buildInboundUpdates([]*common.User{activeUser, inactiveUser})
	update := updates["tag-a"]

	if len(update.accounts) != 1 {
		t.Fatalf("expected 1 active account, got %d", len(update.accounts))
	}
	wantActiveEmail := scopedEmail("666", "tag-a")
	if update.accounts[0].GetEmail() != wantActiveEmail {
		t.Fatalf("active account email: got %q want %q", update.accounts[0].GetEmail(), wantActiveEmail)
	}

	wantRemoveEmail := scopedEmail("777", "tag-a")
	if _, ok := update.removeEmailSet[wantRemoveEmail]; !ok {
		t.Fatalf("removeEmailSet missing scoped email %q, got: %v", wantRemoveEmail, keysOfSet(update.removeEmailSet))
	}
	if _, ok := update.removeEmailSet["777"]; ok {
		t.Fatalf("removeEmailSet should not contain the unscoped email")
	}
}

func keysOf(m map[string]api.Account) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func keysOfSet(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
