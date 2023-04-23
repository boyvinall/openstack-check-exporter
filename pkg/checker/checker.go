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

type CheckerFactory func() (Checker, error)

type Checker interface {
	GetName() string
	Check(ctx context.Context, providerClient *gophercloud.ProviderClient, output *bytes.Buffer) error
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
	providerClient *gophercloud.ProviderClient
	checks         []Checker
	cloud          string
}

func New(cloud string, factories []CheckerFactory) (*CheckManager, error) {
	if cloud == "" {
		cloud = os.Getenv("OS_CLOUD")
	}

	providerClient, err := createProviderClient(cloud)
	if err != nil {
		return nil, err
	}
	providerClient.HTTPClient.Timeout = 10 * time.Second

	cm := &CheckManager{providerClient: providerClient}
	for i := range factories {
		f := factories[i]
		check, err := f()
		if err != nil {
			return nil, err
		}
		cm.checks = append(cm.checks, check)
	}

	return cm, nil
}

func createProviderClient(cloud string) (*gophercloud.ProviderClient, error) {
	if cloud == "" {
		opts, err := openstack.AuthOptionsFromEnv()
		if err != nil {
			return nil, err
		}
		return openstack.AuthenticatedClient(opts)
	}

	return clientconfig.AuthenticatedClient(&clientconfig.ClientOpts{
		Cloud: cloud,
	})
}

func (cm *CheckManager) GetCloud() string {
	return cm.cloud
}

func (cm *CheckManager) Run(ctx context.Context) []*CheckResult {
	count := len(cm.checks)
	results := make([]*CheckResult, 0, count)
	resultCh := make(chan *CheckResult)
	for _, c := range cm.checks {
		check := c // loop invariant
		go func() {
			var output bytes.Buffer
			start := time.Now()
			err := check.Check(ctx, cm.providerClient, &output)
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
	return results
}
