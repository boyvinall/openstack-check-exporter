package horizonlogin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkHorizonLogin struct {
	horizonURL string
	username   string
	password   string
}

// New returns a new Checker instance that logs into Horizon
func New(authOpts *gophercloud.AuthOptions, opts checker.CloudOptions) (checker.Checker, error) {
	return &checkHorizonLogin{}, nil
}

func (c *checkHorizonLogin) GetName() string {
	return "horizon-login"
}

func (c *checkHorizonLogin) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error {
	var u *url.URL
	var err error

	horizonURL := c.horizonURL
	if horizonURL == "" {
		u, err = url.Parse(providerClient.IdentityBase)
		if err != nil {
			return err
		}
		u.Host, _, err = net.SplitHostPort(u.Host)
		if err != nil {
			return err
		}
		u.Path = "/auth/login/"
		horizonURL = u.String()
	} else {
		u, err = url.Parse(horizonURL)
		if err != nil {
			return err
		}
	}

	providerClient.Context = ctx

	resp, err := providerClient.HTTPClient.Get(horizonURL)
	if err != nil {
		return err
	}
	resp.Body.Close()
	fmt.Fprintln(output, resp.Status)
	if resp.StatusCode != http.StatusOK {
		return errors.New("horizon login failed")
	}

	var csrfToken string
	for _, c := range resp.Cookies() {
		if c.Name == "csrftoken" {
			csrfToken = c.Value
		}
	}

	values := url.Values{}
	values.Set("username", c.username)
	values.Set("password", c.password)
	values.Set("csrfmiddlewaretoken", csrfToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, horizonURL, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrftoken", Value: csrfToken})

	resp, err = providerClient.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("horizon login failed")
	}
	fmt.Fprintln(output, resp.Status)
	return nil
}
