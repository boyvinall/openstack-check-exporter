// Package main implements the main entrypoint for the openstack-check-exporter
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
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
	h, err := history.New(400)
	if err != nil {
		return err
	}
	metric := metrics.New()

	// serve http

	errCh := make(chan error)
	go func() {
		http.HandleFunc("/", h.ShowList)
		http.Handle("/detail/", http.StripPrefix("/detail/", http.HandlerFunc(h.ShowDetail)))
		http.Handle("/metrics", promhttp.Handler())
		server := &http.Server{
			Addr:              listenAddress,
			ReadHeaderTimeout: 3 * time.Second,
		}
		slog.Info("serving", "address", listenAddress)
		errCh <- server.ListenAndServe()
	}()

	// run the managers

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	for _, mgr := range managers {

		wg.Add(1)
		go func(m *checker.CheckManager) {
			defer wg.Done()

			// run until the context is cancelled
			slog.Info("running manager",
				"cloud", m.GetCloud(),
			)
			e := m.Run(ctx, func(r checker.CheckResult) bool {
				metric.Update(r)
				h.Append(r)
				h.Trim()
				return false
			})
			if e != nil {
				errCh <- e
				return
			}
		}(mgr)
	}

	// wait for error or signal

	slog.Info("waiting for error or signal")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		os.Interrupt,    // CTRL-C
		syscall.SIGTERM, // e.g. docker graceful shutdown
	)
	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	case s := <-sigs:
		switch s {
		case os.Interrupt:
			slog.Info("Interrupt: CTRL-C")
		case syscall.SIGTERM:
			slog.Info("SIGTERM")
		default:
			slog.Info("Signal: %v", s)
		}
	}
	cancel()

	// ensure we don't exit before all managers are done, otherwise we might leave some stale
	// resources behind (e.g. instances, floating IPs, etc.)
	wg.Wait()

	return err
}

func once(managers []*checker.CheckManager, checks []string) error {
	lock := sync.Mutex{}
	ctx := context.Background()
	for _, mgr := range managers {
		err := mgr.Run(ctx, func(r checker.CheckResult) bool {
			lock.Lock()
			defer lock.Unlock()

			fmt.Println("Start   ", r.Start.UTC().Format(time.RFC3339))
			fmt.Println("Cloud   ", r.Cloud)
			fmt.Println("Name    ", r.Name)
			fmt.Println("Error   ", r.Error)
			fmt.Println("Duration", r.Duration.Truncate(time.Millisecond))
			fmt.Println("Output\n-")
			fmt.Println(r.Output, "\n---")
			return true
		}, checks...)
		if err != nil {
			return err
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
			EnvVars: []string{"OS_CLOUD"},
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
				programLevel := slog.LevelInfo
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
