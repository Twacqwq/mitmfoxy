package main

import "github.com/Twacqwq/mitmfoxy/proxy"

func main() {
	conf := &proxy.Config{
		Addr:         ":8443",
		CertFile:     "../../internal/cert/ca.crt",
		KeyFile:      "../../internal/cert/ca.key",
		UseWebsocket: false,
	}

	mitmfoxy := proxy.New(conf)
	if err := mitmfoxy.Start(); err != nil {
		panic(err)
	}
}
