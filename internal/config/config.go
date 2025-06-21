package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
)

// AddonHost is the public (external) base URL where the addon is accessible.
// It is used for  any links requiring the addon host address.
// It defaults to "http://127.0.0.1:3593" but can be overridden by setting the ADDON_HOST environment variable.
var AddonHost = "http://127.0.0.1:3593"

// ServerListenAddr specifies the network address that the HTTP server will listen on.
// It defaults to ":3593" (all interfaces, TCP port 3593) but can be set with the SERVER_LISTEN_ADDR environment variable.
var ServerListenAddr = ":3593"

func init() {
	if addonHost := os.Getenv("ADDON_HOST"); addonHost != "" {
		u, err := url.Parse(addonHost)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to parse ADDON_HOST: %v", err))
			return
		}
		AddonHost = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	}

	if serverListenAddr := os.Getenv("SERVER_LISTEN_ADDR"); serverListenAddr != "" {
		ServerListenAddr = serverListenAddr
	}
}
