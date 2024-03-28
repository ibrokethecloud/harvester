package main

import (
	"encoding/json"
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/vishvananda/netlink"

	"github.com/harvester/harvester/cmd/network-migration-helper/api"
)

func main() {
	var mappingRequestString string

	flags := []cli.Flag{
		cli.StringFlag{
			Name:        "network-mapping-request",
			EnvVar:      "NETWORK_MAPPING_REQUEST",
			Destination: &mappingRequestString,
			Usage:       "json ended string for mapping request",
			Value:       "",
		},
	}

	app := cli.NewApp()
	app.Name = "harvester network migration helper"
	app.Flags = flags
	app.Action = func(c *cli.Context) error {
		return run(mappingRequestString)
	}
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(mappingRequestString string) error {
	logrus.Info("Starting harvester network helper")

	ctx := signals.SetupSignalContext()
	var mappingRequest []api.NetworkMappingRequest
	err := json.Unmarshal([]byte(mappingRequestString), &mappingRequest)
	if err != nil {
		return fmt.Errorf("error parsing mappingRequest string: %v", err)
	}

	if err := reconcileInterfaces(mappingRequest); err != nil {
		return fmt.Errorf("error reconcilling network mapping requests: %v", err)
	}
	<-ctx.Done()
	return nil
}

func reconcileInterfaces(mappingRequest []api.NetworkMappingRequest) error {
	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("error listing link info: %v", err)
	}

	for _, v := range mappingRequest {
		for _, link := range links {
			if link.Attrs().Name == v.SourceInterface && link.Attrs().Alias != v.AliasName {
				logrus.Errorf("setting up alias name %s for link %s", v.AliasName, v.SourceInterface)
				if err := netlink.LinkSetAlias(link, v.AliasName); err != nil {
					return fmt.Errorf("error creating link alias for %s: %v", v.SourceInterface, err)
				}
			}
		}
	}
	return nil
}
