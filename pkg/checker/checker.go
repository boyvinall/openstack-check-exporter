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

// CheckManager runs a set of checks against an OpenStack cloud
type CheckManager struct {
	authOpts *gophercloud.AuthOptions
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

// Run runs all registered checks in parallel and returns the results
func (cm *CheckManager) Run(ctx context.Context, checks ...string) ([]*CheckResult, error) {
	// First authenticate to get a token - do this only once across all tests.
	// But we consciously create a new client from scratch on each run instead
	// of re-authenticating a client across multiple runs.  This allows us to
	// verify the token workflow more like a real client would.
	providerClient, err := openstack.NewClient(cm.authOpts.IdentityEndpoint)
	if err != nil {
		return nil, err
	}
	providerClient.HTTPClient = newHTTPClient()

	err = openstack.Authenticate(providerClient, *cm.authOpts)
	if err != nil {
		return nil, err
	}

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
	// then run all checks in parallel
	count := len(checksToRun)
	results := make([]*CheckResult, 0, count)
	resultCh := make(chan *CheckResult)
	for _, c := range checksToRun {
		check := c // loop invariant
		go func() {
			var output bytes.Buffer
			start := time.Now()
			err := check.Check(ctx, providerClient, cm.region, &output)
			duration := time.Since(start)
			resultCh <- &CheckResult{
				Cloud:    cm.cloud,
				Name:     check.GetName(),
				Error:    err,
				Start:    start,
				Duration: duration,
				Output:   output.String(),
			}
		}()
	}
	for i := 0; i < count; i++ {
		results = append(results, <-resultCh)
	}
	return results, nil
}
