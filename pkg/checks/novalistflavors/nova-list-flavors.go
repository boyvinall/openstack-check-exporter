// Package novalistflavors implements a `checker.Check` that lists flavors in nova
package novalistflavors

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkNovaListFlavors struct{}

// New returns a new Checker instance that lists flavors in nova
func New(authOpts *gophercloud.AuthOptions, opts checker.CloudOptions) (checker.Checker, error) {
	return &checkNovaListFlavors{}, nil
}

func (c *checkNovaListFlavors) GetName() string {
	return "nova_list_flavors"
}

func (c *checkNovaListFlavors) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error {
	novaClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		return err
	}

	novaClient.Context = ctx

	count := 0
	err = flavors.ListDetail(novaClient, flavors.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		flavorList, e := flavors.ExtractFlavors(page)
		if e != nil {
			return false, e
		}
		for i := range flavorList {
			flavor := &flavorList[i]
			fmt.Fprintln(output, flavor.Name)
		}
		count += len(flavorList)
		return true, nil // true: we list all pages of flavors
	})

	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("no flavors found")
	}

	return nil
}
