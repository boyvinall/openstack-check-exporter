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

type CheckerFactory func(authOpts *gophercloud.AuthOptions, opts CloudOptions) (Checker, error)

type Checker interface {
	GetName() string
	Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error
}

type CheckResult struct {
	Cloud    string
	Name     string
	Error    error
	Start    time.Time
	Duration time.Duration
	Output   string
}

type CheckManager struct {
	authOpts *gophercloud.AuthOptions
	checks   []Checker
	cloud    string
	region   string
}

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

func (cm *CheckManager) GetCloud() string {
	return cm.cloud
}

func (cm *CheckManager) Run(ctx context.Context) ([]*CheckResult, error) {
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

	// then run all checks in parallel
	count := len(cm.checks)
	results := make([]*CheckResult, 0, count)
	resultCh := make(chan *CheckResult)
	for _, c := range cm.checks {
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
