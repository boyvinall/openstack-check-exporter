// Package glancelist implements a `checker.Check` that lists one page glance images to ensure glance is working
package glancelist

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

type checkGlanceList struct{}

// New returns a new Checker instance that lists images in glance
func New(authOpts *gophercloud.AuthOptions, opts checker.CloudOptions) (checker.Checker, error) {
	return &checkGlanceList{}, nil
}

func (c *checkGlanceList) GetName() string {
	return "glance_list_images"
}

func (c *checkGlanceList) Check(ctx context.Context, providerClient *gophercloud.ProviderClient, region string, output *bytes.Buffer) error {

	imageClient, err := openstack.NewImageServiceV2(providerClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		return err
	}
	imageClient.Context = ctx

	count := 0
	err = images.List(imageClient, images.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		imageList, e := images.ExtractImages(page)
		if e != nil {
			return false, e
		}
		for i := range imageList {
			image := &imageList[i]
			fmt.Fprintln(output, image.Name)
		}
		count += len(imageList)
		return false, nil // false: we only list the first page of images
	})

	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("no images found")
	}

	return nil
}
