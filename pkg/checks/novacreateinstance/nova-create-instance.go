// Package novacreateinstance implements a `checker.Check` that creates/deletes a Nova instance
package novacreateinstance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	utilsflavors "github.com/gophercloud/utils/openstack/compute/v2/flavors"
	utilsimages "github.com/gophercloud/utils/openstack/imageservice/v2/images"
	utilsnetworks "github.com/gophercloud/utils/openstack/networking/v2/networks"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkNovaInstance struct {
	serverName  string
	flavorName  string
	imageName   string
	networkName string
}

// New returns a new Checker instance that creates and deletes a Nova instance
func New(authOpts *gophercloud.AuthOptions, opts checker.CloudOptions) (checker.Checker, error) {
	checker := &checkNovaInstance{
		serverName:  "monitoring-test",
		flavorName:  "m1.tiny",
		imageName:   "cirros",
		networkName: "admin-net",
	}
	if _, err := opts.String(checker.GetName(), "server_name", &checker.serverName); err != nil {
		return nil, err
	}
	if _, err := opts.String(checker.GetName(), "flavor_name", &checker.flavorName); err != nil {
		return nil, err
	}
	if _, err := opts.String(checker.GetName(), "image_name", &checker.imageName); err != nil {
		return nil, err
	}
	if _, err := opts.String(checker.GetName(), "network_name", &checker.networkName); err != nil {
		return nil, err
	}

	return &checkNovaInstance{}, nil
}

func (c *checkNovaInstance) GetName() string {
	return "nova-create-instance"
}

func (c *checkNovaInstance) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error {
	novaClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		return err
	}

	novaClient.Context = ctx

	neutronClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		return err
	}
	neutronClient.Context = ctx

	// check the instance doesn't already exist

	allPages, err := servers.List(novaClient, servers.ListOpts{
		Name: c.serverName,
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
	flavorID, err := utilsflavors.IDFromName(novaClient, c.flavorName)
	if err != nil {
		return err
	}
	imageID, err := utilsimages.IDFromName(novaClient, c.imageName)
	if err != nil {
		return err
	}
	networkID, err := utilsnetworks.IDFromName(neutronClient, c.networkName)

	// create the instance

	createOpts := servers.CreateOpts{
		Name:      c.serverName,
		ImageRef:  imageID,
		FlavorRef: flavorID,
		Networks:  []servers.Network{{UUID: networkID}},
	}

	server, err := servers.Create(novaClient, createOpts).Extract()
	if err != nil {
		return err
	}
	serverID := server.ID

	b, err := json.MarshalIndent(server, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(output, string(b))

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
