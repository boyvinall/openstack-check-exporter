package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
	cinderservices "github.com/boyvinall/openstack-check-exporter/pkg/checks/cinder-services"
	glancelist "github.com/boyvinall/openstack-check-exporter/pkg/checks/glance-list"
	glanceshow "github.com/boyvinall/openstack-check-exporter/pkg/checks/glance-show"
	horizonlogin "github.com/boyvinall/openstack-check-exporter/pkg/checks/horizon-login"
	neutronfloatingip "github.com/boyvinall/openstack-check-exporter/pkg/checks/neutron-floating-ip"
	neutronlistnetworks "github.com/boyvinall/openstack-check-exporter/pkg/checks/neutron-list-networks"
	novacreateinstance "github.com/boyvinall/openstack-check-exporter/pkg/checks/nova-create-instance"
	novalistflavors "github.com/boyvinall/openstack-check-exporter/pkg/checks/nova-list-flavors"
	novaservices "github.com/boyvinall/openstack-check-exporter/pkg/checks/nova-services"
	"github.com/boyvinall/openstack-check-exporter/pkg/history"
	"github.com/boyvinall/openstack-check-exporter/pkg/metrics"
)

func serve(managers []*checker.CheckManager) error {
	h, err := history.New(1000)
	if err != nil {
		return err
	}
	m := metrics.New()

	serveCh := make(chan error)
	go func() {
		http.HandleFunc("/", h.ShowList)
		http.Handle("/detail/", http.StripPrefix("/detail/", http.HandlerFunc(h.ShowDetail)))
		http.Handle("/metrics", promhttp.Handler())
		serveCh <- http.ListenAndServe(":8080", nil)
	}()

	ctx := context.Background()
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {

		case err := <-serveCh:
			return err

		case <-ticker.C:
			var latest []*checker.CheckResult
			for _, mgr := range managers {
				results, err := mgr.Run(ctx)
				if err != nil {
					log.Println("ERROR:", err)
				}
				for _, r := range results {
					latest = append(latest, r)
					h.Append(r)
				}
			}
			h.Trim()
			m.Update(latest)
		}
	}
}

func once(managers []*checker.CheckManager) error {
	ctx := context.Background()
	for _, mgr := range managers {
		results, err := mgr.Run(ctx)
		if err != nil {
			return err
		}

		for _, r := range results {
			fmt.Println("Start   ", r.Start.UTC().Format(time.RFC3339))
			fmt.Println("Cloud   ", r.Cloud)
			fmt.Println("Name    ", r.Name)
			fmt.Println("Error   ", r.Error)
			fmt.Println("Duration", r.Duration)
			fmt.Println("Output\n-")
			fmt.Println(r.Output, "\n---")
		}
	}

	return nil
}

func createManagers(settingsFile string, clouds ...string) ([]*checker.CheckManager, error) {
	settings, err := checker.LoadSettingsFromFile(settingsFile)
	if err != nil {
		return nil, err
	}

	var managers []*checker.CheckManager
	for _, cloud := range clouds {
		cloudOpts := settings.GetCloudOptions(cloud)
		mgr, err := checker.New(cloud, cloudOpts, []checker.CheckerFactory{
			glancelist.New,
			glanceshow.New,
			cinderservices.New,
			neutronlistnetworks.New,
			novalistflavors.New,
			neutronfloatingip.New,
			novacreateinstance.New,
			novaservices.New,
			horizonlogin.New,
		})
		if err != nil {
			return nil, err
		}
		managers = append(managers, mgr)
	}
	return managers, nil
}

func main() {
	app := cli.NewApp()
	app.Name = "openstack-check-exporter"
	app.Usage = "Prometheus exporter for OpenStack"
	app.Description = strings.Join([]string{}, "\n")
	app.EnableBashCompletion = true
	app.CommandNotFound = func(c *cli.Context, cmd string) {
		fmt.Printf("ERROR: Unknown command '%s'\n", cmd)
	}
	app.Commands = []*cli.Command{
		{
			Name:        "serve",
			Usage:       "Start the exporter",
			Description: strings.Join([]string{}, "\n"),
			Action: func(c *cli.Context) error {
				managers, err := createManagers(c.String("settings-file"), c.String("cloud"))
				if err != nil {
					return err
				}
				return serve(managers)
			},
		},
		{
			Name:        "once",
			Usage:       "run the checks once and exit",
			Description: strings.Join([]string{}, "\n"),
			Action: func(c *cli.Context) error {
				managers, err := createManagers(c.String("settings-file"), c.String("cloud"))
				if err != nil {
					return err
				}
				return once(managers)
			},
		},
		{
			Name:  "show-cloud-options",
			Usage: "Read settings.yaml and show the resultant options for the given cloud",
			Action: func(c *cli.Context) error {
				settings, err := checker.LoadSettingsFromFile(c.String("settings-file"))
				if err != nil {
					return err
				}
				cloud := c.String("cloud")
				opts := settings.GetCloudOptions(cloud)
				opts.Dump()
				return nil
			},
		},
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "cloud",
			Aliases: []string{"c"},
			Usage:   "OpenStack cloud name",
		},
		&cli.StringFlag{
			Name:    "settings-file",
			Aliases: []string{"f"},
			Usage:   "Path to settings.yaml",
			Value:   "settings.yaml",
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
