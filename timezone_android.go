package main

import (
	"time"

	"github.com/ayn2op/discordo/pkg/tzlocal"
)

// Go always uses UTC as the local time zone on Android.
func init() {
	name, err := tzlocal.Name()
	if err != nil {
		return
	}

	if location, err := time.LoadLocation(name); err == nil {
		time.Local = location
	}
}
