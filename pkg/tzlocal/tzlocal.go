// Package tzlocal gets the IANA name of the system's time zone.
package tzlocal

// Name returns the IANA name of the system's time zone, such as America/New_York.
func Name() (string, error) {
	return system()
}
