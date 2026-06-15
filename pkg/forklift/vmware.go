package forklift

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	corev1 "k8s.io/api/core/v1"
)

type vsphereProvider struct {
	ctx          context.Context
	client       *govmomi.Client
	providerName string
	providerID   string
}

/*
	forklift provider secret is of the format:
stringData:
	user: "username"
	password: "password"
	insecureSkipVerify: "true"
	url: "https://vcenter.example.com/sdk"
	cacert: "CA cert"
*/

func NewVsphereProvider(ctx context.Context, secret *corev1.Secret, providerName, providerID string) (*vsphereProvider, error) {
	var insecure bool
	username, ok := secret.StringData["user"]
	if !ok {
		return nil, fmt.Errorf("no key %q found in secret %s", "user", secret.Name)
	}
	password, ok := secret.StringData["password"]
	if !ok {
		return nil, fmt.Errorf("no key %q found in the secret %s", "password", secret.Name)
	}

	cacert, caCertFound := secret.StringData["cacert"]

	// make sure insecureSkipVerify is true
	insecureSkipVerify, ok := secret.StringData["insecureSkipVerify"]
	if ok {
		insecure = insecureSkipVerify == "true"
	}

	endpoint, ok := secret.StringData["url"]
	if !ok {
		return nil, fmt.Errorf("no key %q found in the secret %s", "url", secret.Name)
	}

	endpointURL, err := url.Parse(string(endpoint))
	if err != nil {
		return nil, fmt.Errorf("error parsing endpoint url: %v", err)
	}

	sc := soap.NewClient(endpointURL, false)

	cfg := tls.Config{}
	if insecure {
		cfg.InsecureSkipVerify = true
	} else {
		cfg.RootCAs = x509.NewCertPool()
		if caCertFound {
			if ok := cfg.RootCAs.AppendCertsFromPEM([]byte(cacert)); !ok {
				return nil, fmt.Errorf("error adding CA cert to pool: %v", err)
			}
		}
	}

	sc.Transport = &http.Transport{
		TLSClientConfig: &cfg,
	}

	vc, err := vim25.NewClient(ctx, sc)
	if err != nil {
		return nil, fmt.Errorf("error creating vim client: %v", err)
	}
	client := &govmomi.Client{
		Client:         vc,
		SessionManager: session.NewManager(vc),
	}

	err = client.Login(ctx, url.UserPassword(string(username), string(password)))
	if err != nil {
		return nil, fmt.Errorf("error during login :%v", err)
	}

	return &vsphereProvider{
		ctx:          ctx,
		client:       client,
		providerName: providerName,
		providerID:   providerID,
	}, nil
}

// GetDatastores returns a list of datastores in the vSphere environment. If datstoreName is not empty, it returns only the datastore with the specified name.
func (v *vsphereProvider) GetDatastores() ([]Datastore, error) {
	manager := view.NewManager(v.client.Client)
	containerView, err := manager.CreateContainerView(v.ctx, v.client.ServiceContent.RootFolder, []string{"Datastore"}, true)
	if err != nil {
		return nil, fmt.Errorf("error creating container view: %v", err)
	}

	defer func() {
		if err := containerView.Destroy(v.ctx); err != nil {
			logrus.Errorf("error destroying container view for endpoint %s: %v", v.providerName, err)
		}
	}()

	var fetchedDS []mo.Datastore
	err = containerView.Retrieve(v.ctx, []string{"Datastore"}, []string{"summary", "info", "browser", "capability", "parent", "host"}, &fetchedDS)
	if err != nil {
		return nil, fmt.Errorf("error retrieving datastores: %v", err)
	}

	var result []Datastore
	for _, ds := range fetchedDS {
		absPath, err := find.InventoryPath(v.ctx, v.client.Client, ds.Self)
		if err != nil {
			return nil, fmt.Errorf("error find absolute path for datastore %s for cluster %s: %v", ds.Summary.Name, v.providerName, err)
		}
		var devices []string
		vmfsInfo, ok := ds.Info.(*types.VmfsDatastoreInfo)
		if ok {
			for _, device := range vmfsInfo.Vmfs.Extent {
				devices = append(devices, device.DiskName)
			}
		}
		parent := ParentObject{
			Kind: ds.Parent.Type,
			ID:   ds.Parent.Value,
		}
		result = append(result, Datastore{
			ID:                 ds.Self.Value,
			Parent:             parent,
			Path:               absPath,
			Revision:           0, // need to find what this is supposed to map to
			Name:               ds.Summary.Name,
			SelfLink:           fmt.Sprintf("providers/vsphere/%s/datastores/%s", v.providerID, ds.Self.Value),
			Type:               ds.Summary.Type,
			Capacity:           ds.Summary.Capacity,
			Free:               ds.Summary.FreeSpace,
			Maintenance:        ds.Summary.MaintenanceMode,
			BackingDeviceNames: devices,
		})
	}

	return result, nil
}

func (v *vsphereProvider) GetNetworks() ([]Network, error) {
	var result []Network
	manager := view.NewManager(v.client.Client)
	containerView, err := manager.CreateContainerView(v.ctx, v.client.ServiceContent.RootFolder, []string{"Network"}, true)
	if err != nil {
		return nil, fmt.Errorf("error creating container view: %v", err)
	}

	defer func() {
		if err := containerView.Destroy(v.ctx); err != nil {
			logrus.Errorf("error destroying container view for endpoint %s: %v", v.providerName, err)
		}
	}()

	var fetchedNetworks []mo.Network
	err = containerView.Retrieve(v.ctx, []string{"Network"}, []string{"summary", "parent"}, &fetchedNetworks)
	if err != nil {
		return nil, fmt.Errorf("error retrieving networks: %v", err)
	}

	for _, network := range fetchedNetworks {
		absPath, err := find.InventoryPath(v.ctx, v.client.Client, network.Self)
		if err != nil {
			return nil, fmt.Errorf("error find absolute path for network %s for cluster %s: %v", network.Name, v.providerName, err)
		}
		parent := ParentObject{
			Kind: network.Parent.Type,
			ID:   network.Parent.Value,
		}
		result = append(result, Network{
			ID:       network.Self.Value,
			Variant:  network.Self.Type,
			Parent:   parent,
			Path:     absPath,
			Revision: 0, // need to find what this is supposed to map to
			Name:     network.Name,
			SelfLink: fmt.Sprintf("providers/vsphere/%s/networks/%s", v.providerID, network.Self.Value), // might need to be changed to a url format
		})
	}

	return result, nil
}

func (v *vsphereProvider) GetVirtualMachines() ([]VirtualMachines, error) {
	var result []VirtualMachines
	manager := view.NewManager(v.client.Client)
	// for debugging we will limit container view to a specific folder, but eventually this should be the entire inventory
	// v.client.ServiceContent.RootFolder
	finder := find.NewFinder(v.client.Client, true)
	folder, err := finder.Folder(v.ctx, "/PG Devlab/vm/gmehta")
	if err != nil {
		return nil, fmt.Errorf("error finding folder: %v", err)
	}
	containerView, err := manager.CreateContainerView(v.ctx, folder.Reference(), []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, fmt.Errorf("error creating container view: %v", err)
	}

	defer func() {
		if err := containerView.Destroy(v.ctx); err != nil {
			logrus.Errorf("error destroying container view for endpoint %s: %v", v.providerName, err)
		}
	}()

	var fetchedVMs []mo.VirtualMachine
	err = containerView.Retrieve(v.ctx, []string{"VirtualMachine"}, []string{"summary", "parent", "storage", "config", "guest", "runtime", "network", "snapshot"}, &fetchedVMs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving virtual machines: %v", err)
	}

	// to help debug object info fetched
	out, err := json.Marshal(fetchedVMs[0])
	if err != nil {
		return nil, fmt.Errorf("error marshaling virtual machines: %v", err)
	}

	err = os.WriteFile("/tmp/vm.json", out, 0644)
	if err != nil {
		return nil, fmt.Errorf("error writing virtual machines to file: %v", err)
	}
	// remove till here

	for _, vm := range fetchedVMs {

		absPath, err := find.InventoryPath(v.ctx, v.client.Client, vm.Self)
		if err != nil {
			return nil, fmt.Errorf("error find absolute path for virtual machine %s for cluster %s: %v", vm.Summary.Config.Name, v.providerName, err)
		}
		var devices []Device
		var networkAttachments []VMNetworkAttachment
		var disks []VMDiskAttachment
		for _, v := range vm.Config.Hardware.Device {
			// devices contains general info about all the devices attached to the vm
			devices = append(devices, Device{
				Kind: v.GetVirtualDevice().DeviceInfo.GetDescription().Label,
			})
			// if device is of type VirtualDisks then we try and identify
			// the info needed to populate the VMDiskAttachment struct
			if disk, ok := v.(*types.VirtualDisk); ok {

			}

			// if device is of type VirtualEthernetCard then we try and identify
			// the info needed to populate the VMNetworkAttachment struct
			if nic, ok := v.(*types.VirtualEthernetCard); ok {
			}
		}

		// identify networks attached to vm object which are part of ManagedObject VirtualMachine definition
		var networksAttached []ParentObject
		for _, v := range vm.Network {
			networksAttached = append(networksAttached, ParentObject{
				Kind: NetworkParentObjectKind,
				ID:   v.Value,
			})
		}

		parent := ParentObject{
			Kind: vm.Parent.Type,
			ID:   vm.Parent.Value,
		}
		result = append(result, VirtualMachines{
			ID:                    vm.Self.Value,
			Parent:                parent,
			Path:                  absPath,
			Revision:              0, // need to find what this is supposed to map to
			Name:                  vm.Summary.Config.Name,
			SelfLink:              fmt.Sprintf("providers/vsphere/%s/virtualmachines/%s", v.providerID, vm.Self.Value),
			LastValidatedRevision: "", // need to find what this is supposed to map to
			IsTemplate:            vm.Summary.Config.Template,
			PowerState:            string(vm.Summary.Runtime.PowerState),
			Host:                  vm.Summary.Runtime.Host.Value,
			Networks:              networksAttached,
		})
	}
	return result, nil
}

func (v *vsphereProvider) Close() error {
	return v.client.Logout(v.ctx)
}

// from vm-import-controller
func (v *vsphereProvider) generateNetworkInfos(devices []types.BaseVirtualDevice) ([]VMNetworkAttachment, error) {

	networks, err := v.GetNetworks()
	if err != nil {
		logrus.Errorf("error fetching networks for provider %s: %v", v.providerName, err)
		return nil, fmt.Errorf("error fetching networks for provider %s: %v", v.providerName, err)
	}

	networkMap := make(map[string]Network)
	for _, network := range networks {
		networkMap[network.ID] = network
	}
	result := make([]VMNetworkAttachment, 0, len(devices))

	for _, d := range devices {
		switch d := d.(type) {
		case *types.VirtualVmxnet:
			obj := d
			summary := identifyNetworkName(networkMap, *obj.GetVirtualDevice())
			if summary == "" {
				summary = obj.DeviceInfo.GetDescription().Summary
			}
			result = append(result, VMNetworkAttachment{
				NetworkName: summary,
				MAC:         obj.MacAddress,
				Model:       migration.NetworkInterfaceModelVirtio,
			})
		case *types.VirtualE1000e:
			obj := d
			summary := identifyNetworkName(networkMap, *obj.GetVirtualDevice())
			if summary == "" {
				summary = obj.DeviceInfo.GetDescription().Summary
			}
			result = append(result, source.NetworkInfo{
				NetworkName: summary,
				MAC:         obj.MacAddress,
				Model:       migration.NetworkInterfaceModelE1000e,
			})
		case *types.VirtualE1000:
			obj := d
			summary := identifyNetworkName(networkMap, *obj.GetVirtualDevice())
			if summary == "" {
				summary = obj.DeviceInfo.GetDescription().Summary
			}
			result = append(result, source.NetworkInfo{
				NetworkName: summary,
				MAC:         obj.MacAddress,
				Model:       migration.NetworkInterfaceModelE1000,
			})
		case *types.VirtualVmxnet3:
			obj := d
			summary := identifyNetworkName(networkMap, *obj.GetVirtualDevice())
			if summary == "" {
				summary = obj.DeviceInfo.GetDescription().Summary
			}
			result = append(result, source.NetworkInfo{
				NetworkName: summary,
				MAC:         obj.MacAddress,
				Model:       migration.NetworkInterfaceModelVirtio,
			})
		case *types.VirtualVmxnet2:
			obj := d
			summary := identifyNetworkName(networkMap, *obj.GetVirtualDevice())
			if summary == "" {
				summary = obj.DeviceInfo.GetDescription().Summary
			}
			result = append(result, source.NetworkInfo{
				NetworkName: summary,
				MAC:         obj.MacAddress,
				Model:       migration.NetworkInterfaceModelVirtio,
			})
		case *types.VirtualPCNet32:
			obj := d
			summary := identifyNetworkName(networkMap, *obj.GetVirtualDevice())
			if summary == "" {
				summary = obj.DeviceInfo.GetDescription().Summary
			}
			result = append(result, source.NetworkInfo{
				NetworkName: summary,
				MAC:         obj.MacAddress,
				Model:       migration.NetworkInterfaceModelPcnet,
			})
		}
	}

	return result
}

// identifyNetworkName uses the backing device for a nic to identify network name correctly
// in case of a nic using a Distributed VSwitch the summary returned from device is of the form
// DVSwitch : HEX NUMBER which breaks network lookup. As a result we need to identify the network name
// from the PortGroupKey
func identifyNetworkName(networkMap map[string]string, device types.VirtualDevice) string {
	var summary string
	backing := device.Backing
	switch b := backing.(type) {
	case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
		obj := b
		logrus.Debugf("looking up portgroupkey: %v", obj.Port.PortgroupKey)
		summary = networkMap[obj.Port.PortgroupKey]
	case *types.VirtualEthernetCardNetworkBackingInfo:
		obj := b
		logrus.Debugf("using devicename: %v", obj.DeviceName)
		summary = obj.DeviceName
	default:
		summary = ""
	}
	return summary
}

func generateVMDiskAttachments(disk types.VirtualDisk) *VMDiskAttachment {
	var unitNumber int32 = 0
	if disk.UnitNumber != nil {
		unitNumber = *disk.UnitNumber
	}

	file := disk.GetVirtualDevice().Backing.(types.BackingVirtualDeviceFileBackingInfo).FileName

	return &VMDiskAttachment{
		Key:                   disk.Key,
		UnitNumber:            unitNumber,
		ControllerKey:         disk.ControllerKey,
		File:                  disk.Backing.GetVirtualDeviceFileBackingInfo().FileName,
		Capacity:              disk.CapacityInBytes,
		Shared:                disk.Shared,
		RawDeviceMapping:      disk.RawDisk != nil,
		Bus:                   disk.Backing.GetVirtualDeviceFileBackingInfo().Datastore.Value, // need to confirm if this is correct
		Mode:                  "",                                                             // need to find what this is supposed to map to
		Serial:                "",                                                             // need to find what this is supposed to map to
		ChangeTrackingenabled: false,                                                          // need to find what this is supposed to map to
	}
}
