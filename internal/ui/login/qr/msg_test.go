package qr

import "testing"

func TestDecodeMessage(t *testing.T) {
	tests := []struct {
		data string
		want any
	}{
		{`{"op":"hello","heartbeat_interval":100,"timeout_ms":200}`, helloMsg{heartbeatInterval: 100, timeoutMS: 200}},
		{`{"op":"nonce_proof","encrypted_nonce":"nonce"}`, nonceProofMsg{encryptedNonce: "nonce"}},
		{`{"op":"pending_remote_init","fingerprint":"fingerprint"}`, pendingRemoteInitMsg{fingerprint: "fingerprint"}},
		{`{"op":"pending_ticket","encrypted_user_payload":"payload"}`, pendingTicketMsg{encryptedUserPayload: "payload"}},
		{`{"op":"cancel"}`, cancelMsg{}},
		{`{"op":"pending_login","ticket":"ticket"}`, pendingLoginMsg{ticket: "ticket"}},
		{`{"op":"unknown"}`, nil},
	}

	for _, test := range tests {
		if got := decodeMessage([]byte(test.data)); got != test.want {
			t.Errorf("decodeMessage(%s) = %#v, want %#v", test.data, got, test.want)
		}
	}
}

func TestDecodeMessageRejectsDuplicateNames(t *testing.T) {
	if _, ok := decodeMessage([]byte(`{"op":"cancel","op":"hello"}`)).(errMsg); !ok {
		t.Fatal("duplicate name was accepted")
	}
}
