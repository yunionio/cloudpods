package aws

import (
	"errors"
	"net/http"
)

func defaultHTTPRedirect(client *http.Client) {
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}

		// use prev Authorization if request has no Authorization
		if req.Header.Get("Authorization") == "" {
			prevAuth := via[len(via)-1].Header.Get("Authorization")
			req.Header.Set("Authorization", prevAuth)
		}

		return nil
	}
}
