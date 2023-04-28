// Package cinderservices implements a `checker.Check` that lists cinder services and checks their state
package cinderservices

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/services"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkCinderServices struct {
}

// New returns a new Checker instance that lists images in glance
func New(authOpts *gophercloud.AuthOptions, opts checker.CloudOptions) (checker.Checker, error) {
	return &checkCinderServices{}, nil
}

func (c *checkCinderServices) GetName() string {
	return "cinder_check_services"
}

func (c *checkCinderServices) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error {

	cinderClient, err := openstack.NewBlockStorageV2(providerClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		return err
	}
	cinderClient.Context = ctx

	healthy := true
	err = services.List(cinderClient, services.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
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
		return fmt.Errorf("cinder services not healthy")
	}

	return err
}
