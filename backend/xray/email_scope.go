package xray

import "github.com/pasarguard/node/backend/xray/api"

// Per-inbound traffic accounting: xray-core's stats module keys counters purely by
// the account's "email" string, globally, regardless of which inbound the traffic
// came through - "user>>>{email}>>>traffic>>>uplink" is the SAME counter no matter
// how many inbounds that email is registered on. Giving the same underlying user a
// distinct, inbound-scoped email per inbound (email does not participate in
// authentication - that's the UUID/password) makes xray-core's existing per-user
// stats counters naturally split out per (user, inbound) pair, with zero protocol
// changes. inboundEmailDelimiter must not be able to appear in a plain numeric user
// id, so a single split(delim, 1) on the panel side unambiguously recovers
// (user_id, inbound_tag) even if the tag itself contains further delimiters.
const inboundEmailDelimiter = ":"

func scopedEmail(email, inboundTag string) string {
	return email + inboundEmailDelimiter + inboundTag
}

// withEmail returns a shallow copy of account with its email overridden - never
// mutates the shared account in place, since the same *XxxAccount pointer (from
// ProxySettings) is reused across every inbound the user is active on.
func withEmail(account api.Account, email string) api.Account {
	switch a := account.(type) {
	case *api.VmessAccount:
		clone := *a
		clone.Email = email
		return &clone

	case *api.VlessAccount:
		clone := *a
		clone.Email = email
		return &clone

	case *api.TrojanAccount:
		clone := *a
		clone.Email = email
		return &clone

	case *api.ShadowsocksTcpAccount:
		clone := *a
		clone.Email = email
		return &clone

	case *api.ShadowsocksAccount:
		clone := *a
		clone.Email = email
		return &clone

	case *api.HysteriaAccount:
		clone := *a
		clone.Email = email
		return &clone

	default:
		return account
	}
}
