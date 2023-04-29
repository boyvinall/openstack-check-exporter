// Package checker provides a framework for running a set of checks against an OpenStack cloud.
package checker

import (
	"bytes"
	"context"
	"os"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
)

// CheckerFactory creates a new `Checker` instance
type CheckerFactory func(authOpts *gophercloud.AuthOptions, opts CloudOptions) (Checker, error) //nolint:revive // checker.CheckerFactory is fine

// Checker is a single check that can be run against an OpenStack cloud.
// E.g. create a network, create a server, etc.
type Checker interface {
	GetName() string
	Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error
}

// CheckResult stores the result from Checker.Check.
// It is used to produce metrics or display results.
type CheckResult struct {
	Cloud    string
	Name     string
	Error    error
	Start    time.Time
	Duration time.Duration
	Output   string
}

// CheckResultCallback is a callback function that is called for each CheckResult.
// If true is returned, then additional checks should be stopped.
type CheckResultCallback func(r CheckResult) bool

// CheckManager runs a set of checks against an OpenStack cloud
type CheckManager struct {
	authOpts *gophercloud.AuthOptions
	opts     CloudOptions
	checks   []Checker
	cloud    string
	region   string
}

// New creates a new CheckManager instance
func New(cloud string, opts CloudOptions, factories []CheckerFactory) (*CheckManager, error) {
	clientopts := clientconfig.ClientOpts{Cloud: cloud}
	authOpts, err := clientconfig.AuthOptions(&clientopts)
	if err != nil {
		return nil, err
	}

	// would be nice if clientconfig.AuthOptions() would somehow return the region to us
	region := os.Getenv("OS_REGION_NAME")
	if cloud != "" {
		var clientConfigCloud *clientconfig.Cloud
		clientConfigCloud, err = clientconfig.GetCloudFromYAML(&clientopts)
		if err != nil {
			return nil, err
		}
		region = clientConfigCloud.RegionName
	}

	cm := &CheckManager{
		authOpts: authOpts,
		opts:     opts,
		cloud:    cloud,
		region:   region,
	}
	for i := range factories {
		checkfactory := factories[i]
		check, err := checkfactory(authOpts, opts)
		if err != nil {
			return nil, err
		}
		cm.checks = append(cm.checks, check)
	}

	return cm, nil
}

// Run runs all registered checks in parallel and calls the callback function for each result
func (cm *CheckManager) Run(ctx context.Context, callback CheckResultCallback, checks ...string) error {

	g := errgroup.Group{}
	for _, c := range cm.getChecksToRun(checks...) {
		check := c // loop invariant

		g.Go(func() error {
			interval := 60
			timeout := interval
			if _, err := cm.opts.Int(check.GetName(), "interval", &interval); err != nil {
				return err
			}
			if _, err := cm.opts.Int(check.GetName(), "timeout", &timeout); err != nil {
				return err
			}
			ticker := time.NewTicker(time.Duration(interval) * time.Second)
			for {
				// Run the check immediately

				slog.Debug("running check",
					"check", check.GetName(),
					"interval", interval,
					"timeout", timeout,
				)

				var output bytes.Buffer
				start := time.Now()

				// We consciously create a new client from scratch on each run instead
				// of re-authenticating a client across multiple runs.  This allows us to
				// verify the token workflow more like a real client would.
				providerClient, err := cm.createAuthenticatedClient()
				if err == nil {
					checkCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
					err = check.Check(checkCtx, providerClient, cm.region, &output)
					cancel()
				}

				// callback even if we failed to create the providerClient
				done := callback(CheckResult{
					Cloud:    cm.cloud,
					Name:     check.GetName(),
					Error:    err,
					Start:    start,
					Duration: time.Since(start),
					Output:   output.String(),
				})
				if done {
					return nil
				}

				// Wait for the next interval, or until the context is done

				select {
				case <-ctx.Done():
					return nil

				case <-ticker.C:
				}
			}
		})
	}
	return g.Wait()
}

func (cm *CheckManager) getChecksToRun(checks ...string) []Checker {
	var checksToRun []Checker
	if len(checks) > 0 {
		checksToRun = make([]Checker, 0, len(checks))
		for _, name := range checks {
			for _, check := range cm.checks {
				if check.GetName() == name {
					checksToRun = append(checksToRun, check)
					break
				}
			}
		}
	} else {
		checksToRun = cm.checks
	}
	return checksToRun
}

func (cm *CheckManager) createAuthenticatedClient() (*gophercloud.ProviderClient, error) {
	providerClient, err := openstack.NewClient(cm.authOpts.IdentityEndpoint)
	if err != nil {
		return nil, err
	}
	providerClient.HTTPClient = newHTTPClient()

	err = openstack.Authenticate(providerClient, *cm.authOpts)
	if err != nil {
		return nil, err
	}

	return providerClient, nil
}
