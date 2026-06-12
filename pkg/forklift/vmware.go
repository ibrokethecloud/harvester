package forklift

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

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
	ctx      context.Context
	client   *govmomi.Client
	endpoint string
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

func NewVsphereProvider(ctx context.Context, secret *corev1.Secret) (*vsphereProvider, error) {
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
		ctx:      ctx,
		client:   client,
		endpoint: endpoint,
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
			logrus.Errorf("error destroying container view for endpoint %s: %v", v.endpoint, err)
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
			return nil, fmt.Errorf("error find absolute path for datastore %s for cluster %s: %v", ds.Summary.Name, v.endpoint, err)
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
			SelfLink:           ds.Self.String(), // might need to be changed to a url format
			Type:               ds.Summary.Type,
			Capacity:           ds.Summary.Capacity,
			Free:               ds.Summary.FreeSpace,
			Maintenance:        ds.Summary.MaintenanceMode,
			BackingDeviceNames: devices,
		})
	}

	return result, nil
}

func (v *vsphereProvider) GetNetworks() {}

func (v *vsphereProvider) GetVirtualMachines() {}

func (v *vsphereProvider) Close() error {
	return v.client.Logout(v.ctx)
}
