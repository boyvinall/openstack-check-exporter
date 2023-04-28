// Package neutronfloatingip implements a `checker.Check` that creates/deletes a floating IP
package neutronfloatingip

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkNeutronFloatingIP struct {
	pool string
}

// New returns a new Checker instance that creates/deletes a floating IP
func New(authOpts *gophercloud.AuthOptions, opts checker.CloudOptions) (checker.Checker, error) {

	c := &checkNeutronFloatingIP{
		pool: "public",
	}
	if _, err := opts.String(c.GetName(), "pool_name", &c.pool); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *checkNeutronFloatingIP) GetName() string {
	return "neutron_floating_ip"
}

func (c *checkNeutronFloatingIP) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error {
	novaClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		return err
	}

	novaClient.Context = ctx

	// Create a floating IP
	floatingIP, err := floatingips.Create(novaClient, floatingips.CreateOpts{
		Pool: c.pool,
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
