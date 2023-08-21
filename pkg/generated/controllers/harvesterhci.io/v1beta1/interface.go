/*
Copyright 2023 Rancher Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by main. DO NOT EDIT.

package v1beta1

import (
	v1beta1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/schemes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	schemes.Register(v1beta1.AddToScheme)
}

type Interface interface {
	Addon() AddonController
	KeyPair() KeyPairController
	Preference() PreferenceController
	SecurityGroup() SecurityGroupController
	Setting() SettingController
	SupportBundle() SupportBundleController
	Upgrade() UpgradeController
	UpgradeLog() UpgradeLogController
	Version() VersionController
	VirtualMachineBackup() VirtualMachineBackupController
	VirtualMachineImage() VirtualMachineImageController
	VirtualMachineRestore() VirtualMachineRestoreController
	VirtualMachineTemplate() VirtualMachineTemplateController
	VirtualMachineTemplateVersion() VirtualMachineTemplateVersionController
}

func New(controllerFactory controller.SharedControllerFactory) Interface {
	return &version{
		controllerFactory: controllerFactory,
	}
}

type version struct {
	controllerFactory controller.SharedControllerFactory
}

func (c *version) Addon() AddonController {
	return NewAddonController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "Addon"}, "addons", true, c.controllerFactory)
}
func (c *version) KeyPair() KeyPairController {
	return NewKeyPairController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "KeyPair"}, "keypairs", true, c.controllerFactory)
}
func (c *version) Preference() PreferenceController {
	return NewPreferenceController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "Preference"}, "preferences", true, c.controllerFactory)
}
func (c *version) SecurityGroup() SecurityGroupController {
	return NewSecurityGroupController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "SecurityGroup"}, "securitygroups", true, c.controllerFactory)
}
func (c *version) Setting() SettingController {
	return NewSettingController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "Setting"}, "settings", false, c.controllerFactory)
}
func (c *version) SupportBundle() SupportBundleController {
	return NewSupportBundleController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "SupportBundle"}, "supportbundles", true, c.controllerFactory)
}
func (c *version) Upgrade() UpgradeController {
	return NewUpgradeController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "Upgrade"}, "upgrades", true, c.controllerFactory)
}
func (c *version) UpgradeLog() UpgradeLogController {
	return NewUpgradeLogController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "UpgradeLog"}, "upgradelogs", true, c.controllerFactory)
}
func (c *version) Version() VersionController {
	return NewVersionController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "Version"}, "versions", true, c.controllerFactory)
}
func (c *version) VirtualMachineBackup() VirtualMachineBackupController {
	return NewVirtualMachineBackupController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "VirtualMachineBackup"}, "virtualmachinebackups", true, c.controllerFactory)
}
func (c *version) VirtualMachineImage() VirtualMachineImageController {
	return NewVirtualMachineImageController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "VirtualMachineImage"}, "virtualmachineimages", true, c.controllerFactory)
}
func (c *version) VirtualMachineRestore() VirtualMachineRestoreController {
	return NewVirtualMachineRestoreController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "VirtualMachineRestore"}, "virtualmachinerestores", true, c.controllerFactory)
}
func (c *version) VirtualMachineTemplate() VirtualMachineTemplateController {
	return NewVirtualMachineTemplateController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "VirtualMachineTemplate"}, "virtualmachinetemplates", true, c.controllerFactory)
}
func (c *version) VirtualMachineTemplateVersion() VirtualMachineTemplateVersionController {
	return NewVirtualMachineTemplateVersionController(schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "VirtualMachineTemplateVersion"}, "virtualmachinetemplateversions", true, c.controllerFactory)
}
