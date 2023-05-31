package checker

import (
	"errors"
	"net/http"
	"time"

	"golang.org/x/exp/slog"
)

// LogRoundTripper satisfies the http.RoundTripper interface and is used to
// customize the default Gophercloud RoundTripper to allow for logging.
type LogRoundTripper struct {
	rt                http.RoundTripper
	numReauthAttempts int
}

// RoundTrip performs a round-trip HTTP request and logs relevant information about it.
func (lrt *LogRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := lrt.rt.RoundTrip(request)
	if response == nil {
		slog.Error("http roundtrip failed",
			"url", request.URL.String(),
			"error", err,
		)
		return nil, err
	}
	slog.Debug("request",
		"url", request.URL.String(),
		"statuscode", response.StatusCode,
	)

	if response.StatusCode == http.StatusUnauthorized {
		if lrt.numReauthAttempts >= 3 {
			return response, errors.New("tried to re-authenticate 3 times with no success")
		}
		lrt.numReauthAttempts++
	}

	return response, nil
}

// newHTTPClient return a custom HTTP client that allows for logging relevant
// information before and after the HTTP request.
func newHTTPClient() http.Client {
	return http.Client{
		Transport: &LogRoundTripper{
			rt: http.DefaultTransport,
		},
		Timeout: 20 * time.Second,
	}
}
