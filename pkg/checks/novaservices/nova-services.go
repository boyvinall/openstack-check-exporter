// Package novaservices implements a `checker.Check` that lists nova services and checks their state
package novaservices

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/services"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkNovaServices struct{}

// New returns a new Checker instance that lists nova services
func New(authOpts *gophercloud.AuthOptions, opts checker.CloudOptions) (checker.Checker, error) {
	return &checkNovaServices{}, nil
}

func (c *checkNovaServices) GetName() string {
	return "nova_check_services"
}

func (c *checkNovaServices) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error {
	novaClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		return err
	}
	novaClient.Context = ctx

	healthy := true
	err = services.List(novaClient, services.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		serviceList, e := services.ExtractServices(page)
		if e != nil {
			healthy = false
			return false, e
		}
		for i := range serviceList {
			s := &serviceList[i]
			if s.Status == "enabled" && s.State != "up" {
				healthy = false
			}
			fmt.Fprintln(output, s.Binary, s.Zone, s.Host, s.State, s.Status, s.DisabledReason)
		}
		return true, nil // true: we want to check all services
	})

	if err != nil {
		return err
	}

	if !healthy {
		return fmt.Errorf("nova services not healthy")
	}

	return nil
}
