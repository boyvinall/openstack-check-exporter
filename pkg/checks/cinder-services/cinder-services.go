package cinderservices

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/services"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkCinderServices struct {
}

// New returns a new Checker instance that lists images in glance
func New() (checker.Checker, error) {
	return &checkCinderServices{}, nil
}

func (c *checkCinderServices) GetName() string {
	return "cinder-list-services"
}

func (c *checkCinderServices) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, output *bytes.Buffer) error {

	cinderClient, err := openstack.NewBlockStorageV2(providerClient, gophercloud.EndpointOpts{Region: os.Getenv("OS_REGION_NAME")})
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
		for _, s := range serviceList {
			if s.Status == "enabled" && s.State != "up" {
				healthy = false
			}
			fmt.Fprintln(output, s.Binary, s.Zone, s.Host, s.State, s.Status, s.DisabledReason)
		}
		return true, nil
	})

	if err != nil {
		return err
	}

	if !healthy {
		return fmt.Errorf("cinder services not healthy")
	}

	return err
}
