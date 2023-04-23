package glanceshow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkGlanceShow struct {
}

// New returns a new Checker instance that lists images in glance
func New() (checker.Checker, error) {
	return &checkGlanceShow{}, nil
}

func (c *checkGlanceShow) GetName() string {
	return "glance-show-image"
}

func (c *checkGlanceShow) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, output *bytes.Buffer) error {

	imageClient, err := openstack.NewImageServiceV2(providerClient, gophercloud.EndpointOpts{Region: os.Getenv("OS_REGION_NAME")})
	if err != nil {
		return err
	}
	imageClient.Context = ctx

	get := images.Get(imageClient, "cirros")
	if get.Err == nil {
		i, err := get.Extract()
		if err != nil {
			return err
		}

		b, err := json.MarshalIndent(i, "", "  ")
		if err != nil {
			return err
		}

		fmt.Fprintln(output, string(b))
	}

	// don't error  if not found .. check for name instead below
	if _, notfound := get.Err.(gophercloud.ErrDefault404); !notfound {
		return get.Err
	}

	err = images.List(imageClient, images.ListOpts{Name: "cirros"}).EachPage(func(page pagination.Page) (bool, error) {
		imageList, e := images.ExtractImages(page)
		if e != nil {
			return false, e
		}
		for i := range imageList {
			image := &imageList[i]
			fmt.Fprintln(output, image.Name)
		}
		return false, nil
	})

	return err
}
