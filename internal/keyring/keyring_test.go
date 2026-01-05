package keyring

import (
	"errors"
	"testing"

	zkeyring "github.com/zalando/go-keyring"
)

const testToken = "abc123"

var testErr = errors.New("boom")

func TestSetToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		zkeyring.MockInit()

		if err := SetToken(testToken); err != nil {
			t.Fatal(err)
		}

		got, err := GetToken()
		if err != nil {
			t.Fatal(err)
		}

		if got != testToken {
			t.Fatalf("got = %q, want = %q", got, testToken)
		}
	})

	t.Run("error", func(t *testing.T) {
		zkeyring.MockInitWithError(testErr)

		if err := SetToken(testToken); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGetToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		zkeyring.MockInit()

		if err := zkeyring.Set(keyringService, keyringUser, testToken); err != nil {
			t.Fatal(err)
		}

		got, err := GetToken()
		if err != nil {
			t.Fatal(err)
		}
		if got != testToken {
			t.Fatalf("got = %q, want = %q", got, testToken)
		}
	})

	t.Run("not found", func(t *testing.T) {
		zkeyring.MockInit()

		_, err := GetToken()
		if !errors.Is(err, zkeyring.ErrNotFound) {
			t.Fatalf("got = %v, want = %v", err, zkeyring.ErrNotFound)
		}
	})

	t.Run("error", func(t *testing.T) {
		zkeyring.MockInitWithError(testErr)

		if _, err := GetToken(); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDeleteToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		zkeyring.MockInit()

		if err := zkeyring.Set(keyringService, keyringUser, testToken); err != nil {
			t.Fatal(err)
		}

		if err := DeleteToken(); err != nil {
			t.Fatal(err)
		}

		_, err := GetToken()
		if !errors.Is(err, zkeyring.ErrNotFound) {
			t.Fatalf("got = %v, want = %v", err, zkeyring.ErrNotFound)
		}
	})

	t.Run("not found", func(t *testing.T) {
		zkeyring.MockInit()

		if err := DeleteToken(); !errors.Is(err, zkeyring.ErrNotFound) {
			t.Fatalf("got = %v, want = %v", err, zkeyring.ErrNotFound)
		}
	})

	t.Run("error", func(t *testing.T) {
		zkeyring.MockInitWithError(testErr)

		if err := DeleteToken(); err == nil {
			t.Fatal("expected error")
		}
	})
}
