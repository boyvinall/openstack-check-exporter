package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boyvinall/openstack-check-exporter/pkg/history"
	"github.com/boyvinall/openstack-check-exporter/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
	cinderservices "github.com/boyvinall/openstack-check-exporter/pkg/checks/cinder-services"
	glancelist "github.com/boyvinall/openstack-check-exporter/pkg/checks/glance-list"
	glanceshow "github.com/boyvinall/openstack-check-exporter/pkg/checks/glance-show"
	neutronlistnetworks "github.com/boyvinall/openstack-check-exporter/pkg/checks/neutron-list-networks"
	novalistflavors "github.com/boyvinall/openstack-check-exporter/pkg/checks/nova-list-flavors"
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
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {

		case err := <-serveCh:
			return err

		case <-ticker.C:
			var latest []*checker.CheckResult
			for _, mgr := range managers {
				results := mgr.Run(ctx)
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
		results := mgr.Run(ctx)

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

func createManagers(clouds ...string) ([]*checker.CheckManager, error) {
	var managers []*checker.CheckManager
	for _, cloud := range clouds {
		mgr, err := checker.New(cloud, []checker.CheckerFactory{
			glancelist.New,
			glanceshow.New,
			cinderservices.New,
			neutronlistnetworks.New,
			novalistflavors.New,
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
			Name:  "serve",
			Usage: "Start the exporter",
			Description: strings.Join([]string{
				"foo",
				"foo",
				"foo",
				"foo",
			}, "\n"),
			Action: func(c *cli.Context) error {
				managers, err := createManagers("")
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
				managers, err := createManagers("")
				if err != nil {
					return err
				}
				return once(managers)
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
