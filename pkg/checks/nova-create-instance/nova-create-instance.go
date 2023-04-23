package novacreateinstance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	utilsflavors "github.com/gophercloud/utils/openstack/compute/v2/flavors"
	utilsimages "github.com/gophercloud/utils/openstack/imageservice/v2/images"
	utilsnetworks "github.com/gophercloud/utils/openstack/networking/v2/networks"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

const serverName = "monitoring-test"

type checkNovaInstance struct{}

// New returns a new Checker instance that creates and deletes a Nova instance
func New() (checker.Checker, error) {
	return &checkNovaInstance{}, nil
}

func (c *checkNovaInstance) GetName() string {
	return "nova-create-instance"
}

func (c *checkNovaInstance) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, output *bytes.Buffer) error {
	novaClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: os.Getenv("OS_REGION_NAME")})
	if err != nil {
		return err
	}

	novaClient.Context = ctx

	neutronClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{Region: os.Getenv("OS_REGION_NAME")})
	if err != nil {
		return err
	}
	neutronClient.Context = ctx

	// check the instance doesn't already exist

	allPages, err := servers.List(novaClient, servers.ListOpts{
		Name: serverName,
	}).AllPages()
	if err != nil {
		return err
	}
	allServers, err := servers.ExtractServers(allPages)
	if err != nil {
		return err
	}
	if len(allServers) > 0 {
		return errors.New("server already exists")
	}

	// get the flavor ID
	flavorID, err := utilsflavors.IDFromName(novaClient, "m1.tiny")
	if err != nil {
		return err
	}
	imageID, err := utilsimages.IDFromName(novaClient, "cirros")
	if err != nil {
		return err
	}
	networkID, err := utilsnetworks.IDFromName(neutronClient, "admin-net")

	// create the instance

	createOpts := servers.CreateOpts{
		Name:      serverName,
		ImageRef:  imageID,
		FlavorRef: flavorID,
		Networks:  []servers.Network{{UUID: networkID}},
	}

	server, err := servers.Create(novaClient, createOpts).Extract()
	if err != nil {
		return err
	}
	serverID := server.ID
	fmt.Fprintln(output, "Created server", server.ID, "in tenant", server.TenantID)

	// wait for the instance to be active
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		if server.Status == "ACTIVE" {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
		}
		server, err = servers.Get(novaClient, serverID).Extract()
		if err != nil {
			return err
		}
	}

	// delete the instance
	err = servers.Delete(novaClient, server.ID).ExtractErr()
	if err != nil {
		return err
	}

	return nil
}
