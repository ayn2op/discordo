package tls

import "github.com/bogdanfinn/tls-client/profiles"

var clientProfile = profiles.Chrome_152

func BrowserName() string {
	return clientProfile.GetClientHelloId().Client
}

func BrowserMajorVersion() string {
	return clientProfile.GetClientHelloId().Version
}

func BrowserVersion() string {
	return BrowserMajorVersion() + ".0.0.0"
}
