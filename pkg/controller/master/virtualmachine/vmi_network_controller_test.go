package virtualmachine

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	k8sfakeclient "k8s.io/client-go/kubernetes/fake"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/harvester/harvester/pkg/generated/clientset/versioned/fake"
	virtualmachinetype "github.com/harvester/harvester/pkg/generated/clientset/versioned/typed/kubevirt.io/v1"
	kv1ctl "github.com/harvester/harvester/pkg/generated/controllers/kubevirt.io/v1"
	"github.com/harvester/harvester/pkg/util/fakeclients"
)

func TestSetDefaultManagementNetworkMacAddress(t *testing.T) {
	type input struct {
		key string
		vmi *kubevirtv1.VirtualMachineInstance
		vm  *kubevirtv1.VirtualMachine
		cr  *appsv1.ControllerRevision
	}
	type output struct {
		vmi *kubevirtv1.VirtualMachineInstance
		vm  *kubevirtv1.VirtualMachine
		err error
	}

	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "ignore nil resource",
			given: input{
				key: "",
				vmi: nil,
				cr:  nil,
			},
			expected: output{
				vmi: nil,
				err: nil,
			},
		},
		{
			name: "ignore deleted resource",
			given: input{
				key: "default/test",
				vmi: &kubevirtv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "default",
						Name:              "test",
						UID:               "fake-vmi-uid",
						DeletionTimestamp: &metav1.Time{},
					},
					Spec: kubevirtv1.VirtualMachineInstanceSpec{},
				},
				cr: &appsv1.ControllerRevision{},
			},
			expected: output{
				vmi: &kubevirtv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "default",
						Name:              "test",
						UID:               "fake-vmi-uid",
						DeletionTimestamp: &metav1.Time{},
					},
					Spec: kubevirtv1.VirtualMachineInstanceSpec{},
				},
				err: nil,
			},
		},
		{
			name: "set mac address",
			given: input{
				key: "default/test",
				vm: &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "default",
						Name:       "test",
						UID:        "fake-vm-uid",
						Generation: 1,
					},
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Networks: []kubevirtv1.Network{
									{
										Name: "default",
										NetworkSource: kubevirtv1.NetworkSource{
											Pod: &kubevirtv1.PodNetwork{},
										},
									},
								},
								Domain: kubevirtv1.DomainSpec{
									Devices: kubevirtv1.Devices{
										Interfaces: []kubevirtv1.Interface{
											{
												Name: "default",
											},
										},
									},
								},
							},
						},
					},
					Status: kubevirtv1.VirtualMachineStatus{
						ObservedGeneration: 1,
					},
				},
				vmi: &kubevirtv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test",
						UID:       "fake-vmi-uid",
					},
					Spec: kubevirtv1.VirtualMachineInstanceSpec{},
					Status: kubevirtv1.VirtualMachineInstanceStatus{
						Interfaces: []kubevirtv1.VirtualMachineInstanceNetworkInterface{
							{
								IP:   "172.16.0.100",
								MAC:  "00:00:00:00:00",
								Name: "default",
							},
							{
								IP:   "172.16.0.101",
								MAC:  "00:01:02:03:04",
								Name: "nic-1",
							},
						},
						Phase:                      kubevirtv1.Running,
						VirtualMachineRevisionName: "test-random-id",
					},
				},
				cr: &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-random-id",
						Namespace: "default",
					},
				},
			},
			expected: output{
				vmi: &kubevirtv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test",
						UID:       "fake-vmi-uid",
					},
					Spec: kubevirtv1.VirtualMachineInstanceSpec{},
					Status: kubevirtv1.VirtualMachineInstanceStatus{
						Interfaces: []kubevirtv1.VirtualMachineInstanceNetworkInterface{
							{
								IP:   "172.16.0.100",
								MAC:  "00:00:00:00:00",
								Name: "default",
							},
							{
								IP:   "172.16.0.101",
								MAC:  "00:01:02:03:04",
								Name: "nic-1",
							},
						},
						Phase:                      kubevirtv1.Running,
						VirtualMachineRevisionName: "test-random-id",
					},
				},
				vm: &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test",
						UID:       "fake-vm-uid",
					},
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Networks: []kubevirtv1.Network{
									{
										Name: "default",
										NetworkSource: kubevirtv1.NetworkSource{
											Pod: &kubevirtv1.PodNetwork{},
										},
									},
								},
								Domain: kubevirtv1.DomainSpec{
									Devices: kubevirtv1.Devices{
										Interfaces: []kubevirtv1.Interface{
											{
												Name:       "default",
												MacAddress: "00:00:00:00:00",
											},
											{
												Name:       "nic-1",
												MacAddress: "00:01:02:03:04",
											},
										},
									},
								},
							},
						},
					},
				},
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		var clientset = fake.NewSimpleClientset()
		var k8sfake = k8sfakeclient.NewSimpleClientset()
		if tc.given.vmi != nil {
			var err = clientset.Tracker().Add(tc.given.vmi)
			assert.Nil(t, err, "mock resource should add into fake controller tracker")
		}
		if tc.given.vm != nil {
			var err = clientset.Tracker().Add(tc.given.vm)
			assert.Nil(t, err, "mock resource should add into fake controller tracker")
		}

		if tc.given.cr != nil && tc.given.vm != nil {
			specString, err := json.Marshal(tc.given.vm.Spec)
			assert.Nil(t, err, "expect no error trying to marshal vmspec")
			tc.given.cr.Data.Raw = specString
			err = k8sfake.Tracker().Add(tc.given.cr)
			assert.Nil(t, err, "mock resource should add into fake controller tracker")
		}

		var ctrl = &VMNetworkController{
			vmClient:  fakeVMClient(clientset.KubevirtV1().VirtualMachines),
			vmCache:   fakeVMCache(clientset.KubevirtV1().VirtualMachines),
			vmiClient: fakeVMIClient(clientset.KubevirtV1().VirtualMachineInstances),
			crCache:   fakeclients.ControllerRevisionCache(k8sfake.AppsV1().ControllerRevisions),
			crClient:  fakeclients.ControllerRevisionClient(k8sfake.AppsV1().ControllerRevisions),
		}

		var actual output
		actual.vmi, actual.err = ctrl.SetDefaultNetworkMacAddress(tc.given.key, tc.given.vmi)
		assert.Nil(t, actual.err, "error during reconcile of SetDefaultNetworkMacAddress %v %s", actual.err, tc.name)
		if tc.given.vmi != nil && tc.given.vm != nil {
			actual.vm, actual.err = ctrl.vmClient.Get(tc.given.vm.Namespace, tc.given.vm.Name, metav1.GetOptions{})
			assert.Nil(t, actual.err, "mock resource should get from fake VM controller, error: %v", actual.err)
			for _, vmIface := range actual.vm.Spec.Template.Spec.Domain.Devices.Interfaces {
				for _, iface := range tc.given.vmi.Status.Interfaces {
					if iface.Name == vmIface.Name {
						assert.Equal(t, iface.MAC, vmIface.MacAddress)
					}
				}
			}
		}

		assert.Equal(t, tc.expected.vmi, actual.vmi, "case %q", tc.name)
	}
}

type fakeVMClient func(string) virtualmachinetype.VirtualMachineInterface

func (c fakeVMClient) Create(vm *kubevirtv1.VirtualMachine) (*kubevirtv1.VirtualMachine, error) {
	return c(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
}

func (c fakeVMClient) Update(vm *kubevirtv1.VirtualMachine) (*kubevirtv1.VirtualMachine, error) {
	return c(vm.Namespace).Update(context.TODO(), vm, metav1.UpdateOptions{})
}

func (c fakeVMClient) UpdateStatus(*kubevirtv1.VirtualMachine) (*kubevirtv1.VirtualMachine, error) {
	panic("implement me")
}

func (c fakeVMClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return c(namespace).Delete(context.TODO(), name, *options)
}

func (c fakeVMClient) Get(namespace, name string, options metav1.GetOptions) (*kubevirtv1.VirtualMachine, error) {
	return c(namespace).Get(context.TODO(), name, options)
}

func (c fakeVMClient) List(namespace string, opts metav1.ListOptions) (*kubevirtv1.VirtualMachineList, error) {
	return c(namespace).List(context.TODO(), opts)
}

func (c fakeVMClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c(namespace).Watch(context.TODO(), opts)
}

func (c fakeVMClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *kubevirtv1.VirtualMachine, err error) {
	return c(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

type fakeVMCache func(string) virtualmachinetype.VirtualMachineInterface

func (c fakeVMCache) Get(namespace, name string) (*kubevirtv1.VirtualMachine, error) {
	return c(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (c fakeVMCache) List(_ string, _ labels.Selector) ([]*kubevirtv1.VirtualMachine, error) {
	panic("implement me")
}

func (c fakeVMCache) AddIndexer(_ string, _ kv1ctl.VirtualMachineIndexer) {
	panic("implement me")
}

func (c fakeVMCache) GetByIndex(_, _ string) ([]*kubevirtv1.VirtualMachine, error) {
	panic("implement me")
}

type fakeVMIClient func(string) virtualmachinetype.VirtualMachineInstanceInterface

func (c fakeVMIClient) Create(vm *kubevirtv1.VirtualMachineInstance) (*kubevirtv1.VirtualMachineInstance, error) {
	return c(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
}

func (c fakeVMIClient) Update(vm *kubevirtv1.VirtualMachineInstance) (*kubevirtv1.VirtualMachineInstance, error) {
	return c(vm.Namespace).Update(context.TODO(), vm, metav1.UpdateOptions{})
}

func (c fakeVMIClient) UpdateStatus(*kubevirtv1.VirtualMachineInstance) (*kubevirtv1.VirtualMachineInstance, error) {
	panic("implement me")
}

func (c fakeVMIClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return c(namespace).Delete(context.TODO(), name, *options)
}

func (c fakeVMIClient) Get(namespace, name string, options metav1.GetOptions) (*kubevirtv1.VirtualMachineInstance, error) {
	return c(namespace).Get(context.TODO(), name, options)
}

func (c fakeVMIClient) List(namespace string, opts metav1.ListOptions) (*kubevirtv1.VirtualMachineInstanceList, error) {
	return c(namespace).List(context.TODO(), opts)
}

func (c fakeVMIClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c(namespace).Watch(context.TODO(), opts)
}

func (c fakeVMIClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *kubevirtv1.VirtualMachineInstance, err error) {
	return c(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}
