// Package horizonlogin implements a `checker.Check` that logs into Horizon
package horizonlogin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gophercloud/gophercloud"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkHorizonLogin struct {
	horizonURL string
	username   string
	password   string
	region     string // used when submitting the login form
	client     *http.Client
}

// New returns a new Checker instance that logs into Horizon
func New(authOpts *gophercloud.AuthOptions, opts checker.CloudOptions) (checker.Checker, error) {
	c := &checkHorizonLogin{
		username: authOpts.Username,
		password: authOpts.Password,
		region:   authOpts.IdentityEndpoint,
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// we disable redirects so we can check for the sessionid cookie
				return http.ErrUseLastResponse
			},
		},
	}

	// read the horizon URL from options
	if _, err := opts.String(c.GetName(), "horizon_url", &c.horizonURL); err != nil {
		return nil, err
	}
	if _, err := opts.String(c.GetName(), "region", &c.region); err != nil {
		return nil, err
	}
	timeout := 10
	if _, err := opts.Int(c.GetName(), "timeout", &timeout); err != nil {
		return nil, err
	}
	c.client.Timeout = time.Duration(timeout) * time.Second

	// if not specified, build the horizon URL from the identity endpoint
	if c.horizonURL == "" {
		u, err := url.Parse(authOpts.IdentityEndpoint)
		if err != nil {
			return nil, err
		}
		u.Host, _, err = net.SplitHostPort(u.Host)
		if err != nil {
			return nil, err
		}
		u.Path = "/auth/login/"
		c.horizonURL = u.String()
	}
	return c, nil
}

func (c *checkHorizonLogin) GetName() string {
	return "horizon-login"
}

func (c *checkHorizonLogin) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error {

	// copy the transport over for logging purposes ... could be a better approach to this
	c.client.Transport = providerClient.HTTPClient.Transport

	// first load the login page to get the CSRF token

	resp, err := c.client.Get(c.horizonURL)
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

	// then post the login form

	values := url.Values{}
	values.Set("username", c.username)
	values.Set("password", c.password)
	values.Set("csrfmiddlewaretoken", csrfToken)
	values.Set("region", c.region)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.horizonURL, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.horizonURL)
	req.AddCookie(&http.Cookie{Name: "csrftoken", Value: csrfToken})

	resp, err = c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check if we got a session cookie - if not then the login failed

	gotSession := false
	for _, c := range resp.Cookies() {
		if c.Name == "sessionid" {
			gotSession = true
		}
	}

	// print result

	fmt.Fprintln(output, resp.Status)
	if !gotSession {
		return errors.New("horizon login failed")
	}

	return nil
}
