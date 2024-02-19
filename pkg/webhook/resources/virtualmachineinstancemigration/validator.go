package virtualmachineinstancemigration

import (
	"encoding/json"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"

	ctlcorev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	ctlv1 "github.com/harvester/harvester/pkg/generated/controllers/kubevirt.io/v1"
	werror "github.com/harvester/harvester/pkg/webhook/error"
	"github.com/harvester/harvester/pkg/webhook/types"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

func NewValidator(podCache ctlcorev1.PodCache, podClient ctlcorev1.PodClient, vmiCache ctlv1.VirtualMachineInstanceCache) types.Validator {
	return &vmimValidator{
		podCache:  podCache,
		podClient: podClient,
		vmiCache:  vmiCache,
	}
}

type vmimValidator struct {
	types.DefaultValidator
	podCache  ctlcorev1.PodCache
	podClient ctlcorev1.PodClient
	vmiCache  ctlv1.VirtualMachineInstanceCache
}

func (v *vmimValidator) Resource() types.Resource {
	return types.Resource{
		Names:      []string{"virtualmachineinstancemigrations"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   kubevirtv1.SchemeGroupVersion.Group,
		APIVersion: kubevirtv1.SchemeGroupVersion.Version,
		ObjectType: &kubevirtv1.VirtualMachineInstanceMigration{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
		},
	}
}

func (v *vmimValidator) Create(_ *types.Request, newObj runtime.Object) error {
	vmimObj := newObj.(*kubevirtv1.VirtualMachineInstanceMigration)
	labelSetMap := map[string]string{
		"vm.kubevirt.io/name": vmimObj.Spec.VMIName,
	}

	podObjs, err := v.podCache.List(vmimObj.Namespace, labels.SelectorFromSet(labelSetMap))
	if err != nil {
		return err
	}
	// no pod found, nothing needed
	if len(podObjs) == 0 {
		return nil
	}

	var running int
	for _, v := range podObjs {
		if v.Status.Phase == corev1.PodRunning {
			running++
		}
	}

	if running > 1 {
		return werror.NewInternalError(fmt.Sprintf("expected to find only 1 running pod for the vmi label but found %d", running))
	}

	vmi, err := v.vmiCache.Get(vmimObj.Namespace, vmimObj.Spec.VMIName)
	if err != nil {
		return err
	}

	updatedPod, err := updatePodAnnotations(podObjs[0], vmi)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(updatedPod.Annotations, podObjs[0].Annotations) {
		logrus.Debugf("virt-launcher pod for vm %s/%s being updated", vmi.Namespace, vmi.Name)
		_, err = v.podClient.Update(updatedPod)
	}
	return err
}

func updatePodAnnotations(pod *corev1.Pod, vmi *kubevirtv1.VirtualMachineInstance) (*corev1.Pod, error) {
	networkMap := namescheme.CreateHashedNetworkNameScheme(vmi.Spec.Networks)
	// networkMap contains newly generated name mapped to kubevirt vmi network name
	// map[default:pod37a8eec1ce1] where default is name of network in vmi spec
	// refer to test cases for more details
	// the multus status annotation contains network name in format of NAD-NS/NAD-NAME
	// the interfaceNameMapping will parse vmi network spec and generate a map which makes
	// it easy to update the multus status annotation
	interfaceNameMapping := make(map[string]string)
	for _, v := range vmi.Spec.Networks {
		newIfName, ok := networkMap[v.Name]
		if ok {
			interfaceNameMapping[v.Multus.NetworkName] = newIfName
		}
	}
	secondaryInterfaceDetails := services.NonDefaultMultusNetworksIndexedByIfaceName(pod)

	// pod does not have ordinal interface names, no further action needed
	if !namescheme.PodHasOrdinalInterfaceName(secondaryInterfaceDetails) {
		return pod, nil
	}

	// pod has ordinal index named interfaces, multus annotation status needs to be patched
	for ifName, val := range secondaryInterfaceDetails {
		if namescheme.OrdinalSecondaryInterfaceName(val.Interface) {
			val.Interface = interfaceNameMapping[val.Name]
			secondaryInterfaceDetails[ifName] = val
		}
	}

	// generate full multus annotation
	ifDetails := MultusNetworksIndexedByIfaceName(pod)
	for k, v := range secondaryInterfaceDetails {
		ifDetails[k] = v
	}

	newNetworkStatus := make([]networkv1.NetworkStatus, len(ifDetails))
	for _, v := range ifDetails {
		newNetworkStatus = append(newNetworkStatus, v)
	}

	newNetworkStatusByte, err := json.Marshal(newNetworkStatus)
	if err != nil {
		return nil, err
	}

	pod.Annotations[networkv1.NetworkStatusAnnot] = string(newNetworkStatusByte)
	return pod, nil
}

func MultusNetworksIndexedByIfaceName(pod *corev1.Pod) map[string]networkv1.NetworkStatus {
	indexedNetworkStatus := map[string]networkv1.NetworkStatus{}
	podNetworkStatus, found := pod.Annotations[networkv1.NetworkStatusAnnot]

	if !found {
		return indexedNetworkStatus
	}

	var networkStatus []networkv1.NetworkStatus
	if err := json.Unmarshal([]byte(podNetworkStatus), &networkStatus); err != nil {
		log.Log.Errorf("failed to unmarshall pod network status: %v", err)
		return indexedNetworkStatus
	}

	for _, ns := range networkStatus {
		indexedNetworkStatus[ns.Interface] = ns
	}

	return indexedNetworkStatus
}
