// Package main implements the main entrypoint for the openstack-check-exporter
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
	"golang.org/x/exp/slog"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
	"github.com/boyvinall/openstack-check-exporter/pkg/checks/cinderservices"
	"github.com/boyvinall/openstack-check-exporter/pkg/checks/glancelist"
	"github.com/boyvinall/openstack-check-exporter/pkg/checks/glanceshow"
	"github.com/boyvinall/openstack-check-exporter/pkg/checks/horizonlogin"
	"github.com/boyvinall/openstack-check-exporter/pkg/checks/neutronfloatingip"
	"github.com/boyvinall/openstack-check-exporter/pkg/checks/neutronlistnetworks"
	"github.com/boyvinall/openstack-check-exporter/pkg/checks/novacreateinstance"
	"github.com/boyvinall/openstack-check-exporter/pkg/checks/novalistflavors"
	"github.com/boyvinall/openstack-check-exporter/pkg/checks/novaservices"
	"github.com/boyvinall/openstack-check-exporter/pkg/history"
	"github.com/boyvinall/openstack-check-exporter/pkg/metrics"
)

func serve(listenAddress string, managers []*checker.CheckManager) error {
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
		server := &http.Server{
			Addr:              listenAddress,
			ReadHeaderTimeout: 3 * time.Second,
		}
		slog.Info("serving", "address", listenAddress)
		serveCh <- server.ListenAndServe()
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

func once(managers []*checker.CheckManager, checks []string) error {
	ctx := context.Background()
	for _, mgr := range managers {
		results, err := mgr.Run(ctx, checks...)
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
				return serve(c.String("listen-address"), managers)
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "listen-address",
					Usage: "Address to listen on for web interface and telemetry",
					Value: ":8080",
				},
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
				return once(managers, c.Args().Slice())
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
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Increase verbosity",
			Action: func(c *cli.Context, verbose bool) error {
				programLevel := slog.LevelWarn
				if verbose {
					programLevel = slog.LevelDebug
				}
				h := slog.HandlerOptions{Level: programLevel}.NewJSONHandler(os.Stderr)
				slog.SetDefault(slog.New(h))
				return nil
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
