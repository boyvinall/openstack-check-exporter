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
	autoDelete  bool
}

// New returns a new Checker instance that creates and deletes a Nova instance
func New(authOpts *gophercloud.AuthOptions, opts checker.CloudOptions) (checker.Checker, error) {
	c := &checkNovaInstance{
		serverName:  "monitoring-test",
		flavorName:  "m1.tiny",
		imageName:   "cirros",
		networkName: "admin-net",
		autoDelete:  false,
	}
	if _, err := opts.String(c.GetName(), "server_name", &c.serverName); err != nil {
		return nil, err
	}
	if _, err := opts.String(c.GetName(), "flavor_name", &c.flavorName); err != nil {
		return nil, err
	}
	if _, err := opts.String(c.GetName(), "image_name", &c.imageName); err != nil {
		return nil, err
	}
	if _, err := opts.String(c.GetName(), "network_name", &c.networkName); err != nil {
		return nil, err
	}
	if _, err := opts.Bool(c.GetName(), "auto_delete", &c.autoDelete); err != nil {
		return nil, err
	}

	if c.serverName == "" {
		return nil, errors.New("server_name must be non-empty")
	}
	if c.flavorName == "" {
		return nil, errors.New("flavor_name must be non-empty")
	}
	if c.imageName == "" {
		return nil, errors.New("image_name must be non-empty")
	}
	if c.networkName == "" {
		return nil, errors.New("network_name must be non-empty")
	}

	return c, nil
}

func (c *checkNovaInstance) GetName() string {
	return "nova_create_instance"
}

func (c *checkNovaInstance) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error {

	// construct our service clients

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

	// resolve names into IDs – do this here, not in the constructor, so that we behave correctly if the IDs change during the lifetime of the checker

	flavorID, err := utilsflavors.IDFromName(novaClient, c.flavorName)
	if err != nil {
		return err
	}
	imageID, err := utilsimages.IDFromName(novaClient, c.imageName)
	if err != nil {
		return err
	}
	networkID, err := utilsnetworks.IDFromName(neutronClient, c.networkName)
	if err != nil {
		return err
	}

	// check the instance doesn't already exist

	allPages, err := servers.List(novaClient, servers.ListOpts{
		Name:   c.serverName,
		Image:  imageID,
		Flavor: flavorID,
	}).AllPages()
	if err != nil {
		return err
	}
	allServers, err := servers.ExtractServers(allPages)
	if err != nil {
		return err
	}
	switch len(allServers) {
	case 0: // all good, go ahead and create it

	case 1:
		if !c.autoDelete {
			return errors.New("server already exists")
		}

		// delete the existing instance
		serverID := allServers[0].ID
		fmt.Fprintln(output, "deleting existing instance", serverID)
		err = servers.Delete(novaClient, serverID).ExtractErr()
		if err != nil {
			return err
		}

	default:
		return errors.New("found multiple servers")
	}

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
