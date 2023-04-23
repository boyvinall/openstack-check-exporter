package neutronlistnetworks

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type neutronListNetworks struct{}

// New returns a new Checker instance that lists networks in neutron
func New() (checker.Checker, error) {
	return &neutronListNetworks{}, nil
}

func (c *neutronListNetworks) GetName() string {
	return "neutron-list-networks"
}

func (c *neutronListNetworks) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, output *bytes.Buffer) error {
	neutronClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{Region: os.Getenv("OS_REGION_NAME")})
	if err != nil {
		return err
	}
	neutronClient.Context = ctx

	healthy := true
	err = networks.List(neutronClient, networks.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		networkList, e := networks.ExtractNetworks(page)
		if e != nil {
			return false, e
		}
		for i := range networkList {
			network := &networkList[i]
			if network.Status != "ACTIVE" && network.AdminStateUp {
				healthy = false
			}
			fmt.Fprintln(output, network.ID, network.Status, network.AdminStateUp, network.Name)
		}
		return true, nil // true: list all networks
	})

	if err != nil {
		return err
	}

	if !healthy {
		return fmt.Errorf("neutron networks not healthy")
	}

	return nil
}
