package glancelist

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkGlanceList struct {
}

// New returns a new Checker instance that lists images in glance
func New() (checker.Checker, error) {
	return &checkGlanceList{}, nil
}

func (c *checkGlanceList) GetName() string {
	return "glance-list-images"
}

func (c *checkGlanceList) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, output *bytes.Buffer) error {

	imageClient, err := openstack.NewImageServiceV2(providerClient, gophercloud.EndpointOpts{Region: os.Getenv("OS_REGION_NAME")})
	if err != nil {
		return err
	}
	imageClient.Context = ctx

	err = images.List(imageClient, images.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
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
