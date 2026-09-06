package qr

import "testing"

func TestDecodeMessage(t *testing.T) {
	tests := []struct {
		name string
		data string
		want any
	}{
		{"hello", `{"op":"hello","heartbeat_interval":100,"timeout_ms":200}`, helloMsg{heartbeatInterval: 100, timeoutMS: 200}},
		{"nonce_proof", `{"op":"nonce_proof","encrypted_nonce":"nonce"}`, nonceProofMsg{encryptedNonce: "nonce"}},
		{"pending_remote_init", `{"op":"pending_remote_init","fingerprint":"fingerprint"}`, pendingRemoteInitMsg{fingerprint: "fingerprint"}},
		{"pending_ticket", `{"op":"pending_ticket","encrypted_user_payload":"payload"}`, pendingTicketMsg{encryptedUserPayload: "payload"}},
		{"cancel", `{"op":"cancel"}`, cancelMsg{}},
		{"pending_login", `{"op":"pending_login","ticket":"ticket"}`, pendingLoginMsg{ticket: "ticket"}},
		{"unknown", `{"op":"unknown"}`, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := decodeMessage([]byte(test.data)); got != test.want {
				t.Errorf("decodeMessage(%s) = %#v, want %#v", test.data, got, test.want)
			}
		})
	}

	t.Run("rejects duplicate names", func(t *testing.T) {
		if _, ok := decodeMessage([]byte(`{"op":"cancel","op":"hello"}`)).(errMsg); !ok {
			t.Fatal("duplicate name was accepted")
		}
	})
}
