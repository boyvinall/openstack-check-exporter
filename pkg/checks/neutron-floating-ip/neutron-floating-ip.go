package neutronfloatingip

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
)

type checkNeutronFloatingIP struct{}

// New returns a new Checker instance that creates/deletes a floating IP
func New() (checker.Checker, error) {
	return &checkNeutronFloatingIP{}, nil
}

func (c *checkNeutronFloatingIP) GetName() string {
	return "neutron-floating-ip"
}

func (c *checkNeutronFloatingIP) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, output *bytes.Buffer) error {
	novaClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: os.Getenv("OS_REGION_NAME")})
	if err != nil {
		return err
	}

	novaClient.Context = ctx

	// Create a floating IP
	floatingIP, err := floatingips.Create(novaClient, floatingips.CreateOpts{
		Pool: os.Getenv("OS_POOL_NAME"),
	}).Extract()
	if err != nil {
		return err
	}
	fmt.Fprintln(output, "Created floating IP", floatingIP.ID, floatingIP.IP)

	// Delete the floating IP
	err = floatingips.Delete(novaClient, floatingIP.ID).ExtractErr()
	if err != nil {
		return err
	}
	fmt.Fprintln(output, "Deleted floating IP", floatingIP.ID)
	return nil
}
