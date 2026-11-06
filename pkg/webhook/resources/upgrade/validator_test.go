package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/api/core/v1"

	harvesterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	fake "github.com/harvester/harvester/pkg/generated/clientset/versioned/fake"
	"github.com/harvester/harvester/pkg/util"
	"github.com/harvester/harvester/pkg/util/drainhelper"
	"github.com/harvester/harvester/pkg/util/fakeclients"
)

const (
	node1    = "node1"
	node2    = "node2"
	machine1 = "machine1"
	machine2 = "machine2"
)

func TestUpgradeValidator_validatePauseMapAnnotation(t *testing.T) {
	givenNodes := []*corev1.Node{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-0",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
			},
		},
	}

	var testCases = []struct {
		name      string
		upgrade   *harvesterv1.Upgrade
		expectErr bool
	}{
		{
			name: "empty string",
			upgrade: &harvesterv1.Upgrade{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						util.AnnotationNodeUpgradePauseMap: "",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "bad json string",
			upgrade: &harvesterv1.Upgrade{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						util.AnnotationNodeUpgradePauseMap: "{\"node-0\"}",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "empty pause map",
			upgrade: &harvesterv1.Upgrade{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						util.AnnotationNodeUpgradePauseMap: "{}",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "unpause nodes in pause map",
			upgrade: &harvesterv1.Upgrade{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						util.AnnotationNodeUpgradePauseMap: "{\"node-0\":\"unpause\",\"node-1\":\"pause\"}",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid desire action for nodes in pause map",
			upgrade: &harvesterv1.Upgrade{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						util.AnnotationNodeUpgradePauseMap: "{\"node-0\":\"pause\",\"node-1\":\"restart\"}",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "nodes in pause map are a subset of cluster nodes",
			upgrade: &harvesterv1.Upgrade{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						util.AnnotationNodeUpgradePauseMap: "{\"node-0\":\"pause\"}",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "some nodes in pause map do not exist",
			upgrade: &harvesterv1.Upgrade{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						util.AnnotationNodeUpgradePauseMap: "{\"node-0\":\"pause\",\"node-100\":\"pause\"}",
					},
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodes := make([]runtime.Object, 0, len(givenNodes))
			for _, node := range givenNodes {
				nodes = append(nodes, node)
			}
			clientset := fake.NewSimpleClientset(nodes...)
			validator := &upgradeValidator{
				nodes: fakeclients.NodeCache(clientset.CoreV1().Nodes),
			}

			err := validator.validatePauseMapAnnotation(tc.upgrade)

			assert.Equal(t, tc.expectErr, err != nil, tc.name)
		})
	}
}

func getTestAddon(enabled bool) *harvesterv1.Addon {
	return &harvesterv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name: "addon1",
		},
		Spec: harvesterv1.AddonSpec{
			Repo:          "repo1",
			Chart:         "chart1",
			Version:       "version1",
			Enabled:       enabled,
			ValuesContent: "sample",
		},
	}
}

func setAddonState(addon *harvesterv1.Addon, state harvesterv1.AddonState) {
	if addon == nil {
		return
	}
	addon.Status.Status = state
}

func Test_validateAddons(t *testing.T) {
	var testCases = []struct {
		name          string
		enabled       bool
		addonState    harvesterv1.AddonState
		expectedError bool
	}{
		{
			name:          "good: addon is enabled, and deployed",
			enabled:       true,
			expectedError: false,
			addonState:    harvesterv1.AddonDeployed,
		},
		{
			name:          "bad: addon is enabled, but with no state",
			enabled:       true,
			expectedError: true,
		},
		{
			name:          "bad: addon is enabled, but with enabling state",
			enabled:       true,
			expectedError: true,
			addonState:    harvesterv1.AddonEnabling,
		},
		{
			name:          "bad: addon is enabled, but with init state",
			enabled:       true,
			expectedError: true,
			addonState:    harvesterv1.AddonInitState,
		},
		{
			name:          "bad: addon is enabled, but with updating state",
			enabled:       true,
			expectedError: true,
			addonState:    harvesterv1.AddonUpdating,
		},
		{
			name:          "bad: addon is enabled, but with invalid state",
			enabled:       true,
			expectedError: true,
			addonState:    "invalid",
		},
		{
			name:          "good: addon is disabled, no state, it is from initial",
			enabled:       false,
			expectedError: false,
		},
		{
			name:          "good: addon is disabled, and with disabled state",
			enabled:       false,
			expectedError: false,
			addonState:    harvesterv1.AddonDisabled,
		},
		{
			name:          "bad: addon is disabled, but with disabling state",
			enabled:       false,
			expectedError: true,
			addonState:    harvesterv1.AddonDisabling,
		},
		{
			name:          "bad: addon is disabled, but with invalid state",
			enabled:       false,
			expectedError: true,
			addonState:    "invalid",
		},
	}

	for _, tc := range testCases {
		clientset := fake.NewSimpleClientset()
		fakeAddonCache := fakeclients.AddonCache(clientset.HarvesterhciV1beta1().Addons)

		validator := NewValidator(nil, fakeAddonCache, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).(*upgradeValidator)
		addon := getTestAddon(tc.enabled)
		if tc.addonState != "" {
			setAddonState(addon, tc.addonState)
		}
		err := clientset.Tracker().Add(addon)
		assert.Nil(t, err)

		err = validator.checkAddons()
		if tc.expectedError {
			assert.NotNil(t, err, tc.name)
		} else {
			assert.Nil(t, err, tc.name)
		}
	}
}

func TestUpgradeValidator_checkStaleHotplugVolumes(t *testing.T) {
	newNode := func(name string) *corev1.Node {
		return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}
	}
	hotplugVolume := func(name, claimName string) kubevirtv1.Volume {
		return kubevirtv1.Volume{
			Name: name,
			VolumeSource: kubevirtv1.VolumeSource{
				PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: claimName,
					},
					Hotpluggable: true,
				},
			},
		}
	}
	newVM := func(name string, volumeRequests []kubevirtv1.VirtualMachineVolumeRequest, volumes ...kubevirtv1.Volume) *kubevirtv1.VirtualMachine {
		return &kubevirtv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: kubevirtv1.VirtualMachineSpec{
				Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
					Spec: kubevirtv1.VirtualMachineInstanceSpec{Volumes: volumes},
				},
			},
			Status: kubevirtv1.VirtualMachineStatus{VolumeRequests: volumeRequests},
		}
	}
	newVMI := func(name string, phase kubevirtv1.VirtualMachineInstancePhase, volumes ...kubevirtv1.Volume) *kubevirtv1.VirtualMachineInstance {
		return &kubevirtv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec:       kubevirtv1.VirtualMachineInstanceSpec{Volumes: volumes},
			Status:     kubevirtv1.VirtualMachineInstanceStatus{Phase: phase},
		}
	}

	testCases := []struct {
		name        string
		nodes       []*corev1.Node
		vms         []*kubevirtv1.VirtualMachine
		vmis        []*kubevirtv1.VirtualMachineInstance
		expectError bool
		errorKey    string
	}{
		{
			name:  "no stale hotplug state",
			nodes: []*corev1.Node{newNode(node1), newNode(node2)},
			vms:   []*kubevirtv1.VirtualMachine{newVM("vm1", nil)},
			vmis:  []*kubevirtv1.VirtualMachineInstance{newVMI("vm1", kubevirtv1.Running)},
		},
		{
			name:  "VM with pending volumeRequests is rejected",
			nodes: []*corev1.Node{newNode(node1), newNode(node2)},
			vms: []*kubevirtv1.VirtualMachine{
				newVM("vm1", []kubevirtv1.VirtualMachineVolumeRequest{
					{AddVolumeOptions: &kubevirtv1.AddVolumeOptions{Name: "disk-1"}},
				}),
			},
			expectError: true,
			errorKey:    "pending volume hotplug requests",
		},
		{
			name:  "single-node cluster skips the check",
			nodes: []*corev1.Node{newNode(node1)},
			vms: []*kubevirtv1.VirtualMachine{
				newVM("vm1", []kubevirtv1.VirtualMachineVolumeRequest{
					{AddVolumeOptions: &kubevirtv1.AddVolumeOptions{Name: "disk-1"}},
				}),
			},
		},
		{
			name:  "running VMI with hotplug volume missing from VM spec is rejected",
			nodes: []*corev1.Node{newNode(node1), newNode(node2)},
			vms:   []*kubevirtv1.VirtualMachine{newVM("vm1", nil)},
			vmis: []*kubevirtv1.VirtualMachineInstance{
				newVMI("vm1", kubevirtv1.Running, hotplugVolume("disk-1", "pvc-1")),
			},
			expectError: true,
			errorKey:    "exist in the VMI but not in the VM spec",
		},
		{
			name:  "running VMI with hotplug volume present in VM spec is allowed",
			nodes: []*corev1.Node{newNode(node1), newNode(node2)},
			vms:   []*kubevirtv1.VirtualMachine{newVM("vm1", nil, hotplugVolume("disk-1", "pvc-1"))},
			vmis: []*kubevirtv1.VirtualMachineInstance{
				newVMI("vm1", kubevirtv1.Running, hotplugVolume("disk-1", "pvc-1")),
			},
		},
		{
			name:  "non-running VMI with stale hotplug volume is allowed",
			nodes: []*corev1.Node{newNode(node1), newNode(node2)},
			vms:   []*kubevirtv1.VirtualMachine{newVM("vm1", nil)},
			vmis: []*kubevirtv1.VirtualMachineInstance{
				newVMI("vm1", kubevirtv1.Succeeded, hotplugVolume("disk-1", "pvc-1")),
			},
		},
		{
			name:  "standalone VMI without a VM is ignored",
			nodes: []*corev1.Node{newNode(node1), newNode(node2)},
			vmis: []*kubevirtv1.VirtualMachineInstance{
				newVMI("vmi-only", kubevirtv1.Running, hotplugVolume("disk-1", "pvc-1")),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0, len(tc.nodes)+len(tc.vms)+len(tc.vmis))
			for _, node := range tc.nodes {
				objs = append(objs, node)
			}
			for _, vm := range tc.vms {
				objs = append(objs, vm)
			}
			for _, vmi := range tc.vmis {
				objs = append(objs, vmi)
			}
			clientset := fake.NewSimpleClientset(objs...)
			validator := &upgradeValidator{
				nodes:    fakeclients.NodeCache(clientset.CoreV1().Nodes),
				vmCache:  fakeclients.VirtualMachineCache(clientset.KubevirtV1().VirtualMachines),
				vmiCache: fakeclients.VirtualMachineInstanceCache(clientset.KubevirtV1().VirtualMachineInstances),
			}

			err := validator.checkStaleHotplugVolumes()

			if tc.expectError {
				assert.NotNil(t, err, tc.name)
				assert.Contains(t, err.Error(), tc.errorKey, tc.name)
			} else {
				assert.Nil(t, err, tc.name)
			}
		})
	}
}

func TestUpgradeValidator_checkNodes_PendingDrainRequest(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
		},
	}
	drainhelper.SetDrainRequest(node, false)

	clientset := fake.NewSimpleClientset(node)
	validator := &upgradeValidator{
		nodes: fakeclients.NodeCache(clientset.CoreV1().Nodes),
	}
	upgrade := &harvesterv1.Upgrade{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				util.AnnotationSkipGarbageCollectionThresholdCheck: util.AnnotationValueTrue,
			},
		},
	}

	err := validator.checkNodes(upgrade)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pending maintenance mode request")
}

func TestUpgradeValidator_checkNodes_MaintenanceMode(t *testing.T) {
	tests := []struct {
		name        string
		status      corev1.ConditionStatus
		reason      string
		expectError bool
	}{
		{
			name:        "no condition",
			expectError: false,
		},
		{
			name:        "validating blocks upgrade",
			status:      corev1.ConditionTrue,
			reason:      util.NodeConditionReasonValidating,
			expectError: true,
		},
		{
			name:        "completed blocks upgrade",
			status:      corev1.ConditionTrue,
			reason:      util.NodeConditionReasonCompleted,
			expectError: true,
		},
		{
			name:        "false error blocks upgrade",
			status:      corev1.ConditionFalse,
			reason:      util.NodeConditionReasonError,
			expectError: true,
		},
		{
			name:        "true error blocks upgrade",
			status:      corev1.ConditionTrue,
			reason:      util.NodeConditionReasonError,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
			if tc.reason != "" {
				util.SetMaintenanceModeCondition(node, tc.status, tc.reason, "test")
			}

			clientset := fake.NewSimpleClientset(node)
			validator := &upgradeValidator{
				nodes: fakeclients.NodeCache(clientset.CoreV1().Nodes),
			}

			// Skip garbage collection threshold check so checkDiskSpace is not invoked
			upgrade := &harvesterv1.Upgrade{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						util.AnnotationSkipGarbageCollectionThresholdCheck: util.AnnotationValueTrue,
					},
				},
			}

			err := validator.checkNodes(upgrade)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "disable or clear maintenance mode before upgrading")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
