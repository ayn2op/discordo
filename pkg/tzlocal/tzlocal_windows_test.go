package tzlocal

import (
	"testing"
	"time"
)

func TestSystem(t *testing.T) {
	name, err := system()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := time.LoadLocation(name); err != nil {
		t.Errorf("%q is not a zone: %v", name, err)
	}
}
