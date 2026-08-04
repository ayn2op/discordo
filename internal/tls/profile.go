package tls

import "github.com/bogdanfinn/tls-client/profiles"

var clientProfile = profiles.Chrome_146

func BrowserName() string {
	return clientProfile.GetClientHelloId().Client
}

func BrowserVersion() string {
	return clientProfile.GetClientHelloId().Version + ".0.0.0"
}
