package pod

import (
	"encoding/json"
	"fmt"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"kubevirt.io/kubevirt/pkg/network/namescheme"

	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/sirupsen/logrus"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	networkhelperapi "github.com/harvester/harvester/cmd/network-migration-helper/api"
	"github.com/harvester/harvester/pkg/generated/controllers/harvesterhci.io/v1beta1"
	v1 "github.com/harvester/harvester/pkg/generated/controllers/kubevirt.io/v1"
	"github.com/harvester/harvester/pkg/util"
	"github.com/harvester/harvester/pkg/webhook/types"
)

var matchingLabels = []labels.Set{
	{
		"longhorn.io/component": "backing-image-data-source",
	},
	{
		"app.kubernetes.io/name":      "harvester",
		"app.kubernetes.io/component": "apiserver",
	},
	{
		"app": "rancher",
	},
}

var vmMatchingLabels = []labels.Set{
	{
		"kubevirt.io": "virt-launcher",
	},
}

func NewMutator(settingCache v1beta1.SettingCache, vmiCache v1.VirtualMachineInstanceCache) types.Mutator {
	return &podMutator{
		setttingCache: settingCache,
		vmiCache:      vmiCache,
	}
}

// podMutator injects Harvester settings like http proxy envs and trusted CA certs to system pods that may access
// external services. It includes harvester apiserver and longhorn backing-image-data-source pods.
type podMutator struct {
	types.DefaultMutator
	setttingCache v1beta1.SettingCache
	vmiCache      v1.VirtualMachineInstanceCache
}

func newResource(ops []admissionregv1.OperationType) types.Resource {
	return types.Resource{
		Names:          []string{string(corev1.ResourcePods)},
		Scope:          admissionregv1.NamespacedScope,
		APIGroup:       corev1.SchemeGroupVersion.Group,
		APIVersion:     corev1.SchemeGroupVersion.Version,
		ObjectType:     &corev1.Pod{},
		OperationTypes: ops,
	}
}

func (m *podMutator) Resource() types.Resource {
	return newResource([]admissionregv1.OperationType{
		admissionregv1.Create,
	})
}

func (m *podMutator) Create(_ *types.Request, newObj runtime.Object) (types.PatchOps, error) {
	pod := newObj.(*corev1.Pod)

	if IsHarvesterCorePod(pod) {
		var patchOps types.PatchOps
		httpProxyPatches, err := m.httpProxyPatches(pod)
		if err != nil {
			return nil, err
		}
		patchOps = append(patchOps, httpProxyPatches...)
		additionalCAPatches, err := m.additionalCAPatches(pod)
		if err != nil {
			return nil, err
		}
		patchOps = append(patchOps, additionalCAPatches...)

		return patchOps, nil
	}

	if IsKubevirtLauncherPod(pod) {
		logrus.Debugf("found virt-launcher pod %s/%s", pod.Namespace, pod.GenerateName)
		multusPatch, err := m.multusAnnotationPatch(pod)
		if err != nil {
			return nil, err
		}
		return multusPatch, nil
	}

	return nil, nil

}

func IsHarvesterCorePod(pod *corev1.Pod) bool {
	podLabels := labels.Set(pod.Labels)
	var match bool
	for _, v := range matchingLabels {
		if v.AsSelector().Matches(podLabels) {
			match = true
			break
		}
	}
	return match
}

func IsKubevirtLauncherPod(pod *corev1.Pod) bool {
	podLabels := labels.Set(pod.Labels)
	var match bool
	for _, v := range vmMatchingLabels {
		if v.AsSelector().Matches(podLabels) {
			match = true
			break
		}
	}
	return match
}

func (m *podMutator) httpProxyPatches(pod *corev1.Pod) (types.PatchOps, error) {
	proxySetting, err := m.setttingCache.Get("http-proxy")
	if err != nil || proxySetting.Value == "" {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var httpProxyConfig util.HTTPProxyConfig
	if err := json.Unmarshal([]byte(proxySetting.Value), &httpProxyConfig); err != nil {
		return nil, err
	}
	if httpProxyConfig.HTTPProxy == "" && httpProxyConfig.HTTPSProxy == "" && httpProxyConfig.NoProxy == "" {
		return nil, nil
	}

	var proxyEnvs = []corev1.EnvVar{
		{
			Name:  util.HTTPProxyEnv,
			Value: httpProxyConfig.HTTPProxy,
		},
		{
			Name:  util.HTTPSProxyEnv,
			Value: httpProxyConfig.HTTPSProxy,
		},
		{
			Name:  util.NoProxyEnv,
			Value: util.AddBuiltInNoProxy(httpProxyConfig.NoProxy),
		},
	}
	var patchOps types.PatchOps
	for idx, container := range pod.Spec.Containers {
		envPatches, err := envPatches(container.Env, proxyEnvs, fmt.Sprintf("/spec/containers/%d/env", idx))
		if err != nil {
			return nil, err
		}
		patchOps = append(patchOps, envPatches...)
	}
	return patchOps, nil
}

func envPatches(target, envVars []corev1.EnvVar, basePath string) (types.PatchOps, error) {
	var (
		patchOps types.PatchOps
		value    interface{}
		path     string
		first    = len(target) == 0
	)
	for _, envVar := range envVars {
		if first {
			first = false
			path = basePath
			value = []corev1.EnvVar{envVar}
		} else {
			path = basePath + "/-"
			value = envVar
		}
		valueStr, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		patchOps = append(patchOps, fmt.Sprintf(`{"op": "add", "path": "%s", "value": %s}`, path, valueStr))
	}
	return patchOps, nil
}

func (m *podMutator) additionalCAPatches(pod *corev1.Pod) (types.PatchOps, error) {
	additionalCASetting, err := m.setttingCache.Get("additional-ca")
	if err != nil || additionalCASetting.Value == "" {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var (
		additionalCAvolume = corev1.Volume{
			Name: "additional-ca-volume",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: pointer.Int32(400),
					SecretName:  util.AdditionalCASecretName,
				},
			},
		}
		additionalCAVolumeMount = corev1.VolumeMount{
			Name:      "additional-ca-volume",
			MountPath: "/etc/ssl/certs/" + util.AdditionalCAFileName,
			SubPath:   util.AdditionalCAFileName,
			ReadOnly:  true,
		}
		patchOps types.PatchOps
	)

	volumePatch, err := volumePatch(pod.Spec.Volumes, additionalCAvolume)
	if err != nil {
		return nil, err
	}
	patchOps = append(patchOps, volumePatch)

	for idx, container := range pod.Spec.Containers {
		volumeMountPatch, err := volumeMountPatch(container.VolumeMounts, fmt.Sprintf("/spec/containers/%d/volumeMounts", idx), additionalCAVolumeMount)
		if err != nil {
			return nil, err
		}
		patchOps = append(patchOps, volumeMountPatch)
	}

	return patchOps, nil
}

func volumePatch(target []corev1.Volume, volume corev1.Volume) (string, error) {
	var (
		value      interface{} = []corev1.Volume{volume}
		path                   = "/spec/volumes"
		first                  = len(target) == 0
		valueBytes []byte
		err        error
	)
	if !first {
		value = volume
		path = path + "/-"
	}
	valueBytes, err = json.Marshal(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"op": "add", "path": "%s", "value": %s}`, path, valueBytes), nil
}

func volumeMountPatch(target []corev1.VolumeMount, path string, volumeMount corev1.VolumeMount) (string, error) {
	var (
		value interface{} = []corev1.VolumeMount{volumeMount}
		first             = len(target) == 0
	)
	if !first {
		path = path + "/-"
		value = volumeMount
	}
	valueStr, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{"op": "add", "path": "%s", "value": %s}`, path, valueStr), nil
}

func (m *podMutator) multusAnnotationPatch(pod *corev1.Pod) (types.PatchOps, error) {
	// check if pod has multus annotation for non default multus networks with ordinal interface names
	// patch is only needed in such cases to convert interfaces from ordinal to hashed interface names
	// if pod already has a hashed interface name then no further action is needed
	logrus.Info(pod.Annotations)
	owned, vmiName := podOwnedByVMI(pod)
	if owned {
		vmi, err := m.vmiCache.Get(pod.Namespace, vmiName)
		if err != nil {
			return nil, err
		}
		return generateMultusAnnotationPatch(vmi, pod)
	}
	logrus.Debugf("no ordinal interface names found, skipping pod %s/%s", pod.Namespace, pod.GenerateName)
	return nil, nil
}

func podOwnedByVMI(pod *corev1.Pod) (bool, string) {
	for _, v := range pod.OwnerReferences {
		if v.APIVersion == "kubevirt.io/v1" && v.Kind == "VirtualMachineInstance" {
			return true, v.Name
		}
	}
	return false, ""
}

func generateMultusAnnotationPatch(vmi *kubevirtv1.VirtualMachineInstance, pod *corev1.Pod) (types.PatchOps, error) {
	var patchOps types.PatchOps
	logrus.Debugf("checking if pod for vmi %s/%s needs mutation", vmi.Namespace, vmi.Name)
	networkMap := namescheme.CreateHashedNetworkNameScheme(vmi.Spec.Networks)

	currentNetworkRequest := pod.Annotations[networkv1.NetworkAttachmentAnnot]
	networkDefs := []networkv1.NetworkSelectionElement{}
	err := json.Unmarshal([]byte(currentNetworkRequest), &networkDefs)
	if err != nil {
		return patchOps, err
	}

	macDetails := generateNetworkMacMap(vmi)
	// networkMap contains a map of vmi network name and generated pod name
	// this needs to be mapped to multus network name as well to ensure patch
	// can be generated
	vmiNetworkPodMap := make(map[string]string)
	for _, v := range vmi.Spec.Networks {
		podIfName, ok := networkMap[v.Name]
		if ok {
			vmiNetworkPodMap[v.Multus.NetworkName] = podIfName
		}
	}

	var aliasRequest []networkhelperapi.NetworkMappingRequest
	// rename network interfaces if needed
	for i := range networkDefs {
		networkName := fmt.Sprintf("%s/%s", networkDefs[i].Namespace, networkDefs[i].Name)
		podIfName, ok := vmiNetworkPodMap[networkName]
		if ok && namescheme.OrdinalSecondaryInterfaceName(networkDefs[i].InterfaceRequest) {
			aliasRequest = append(aliasRequest, networkhelperapi.NetworkMappingRequest{
				SourceInterface: fmt.Sprintf("tap%s", podIfName[3:]),
				AliasName:       fmt.Sprintf("tap%s", networkDefs[i].InterfaceRequest[3:]),
			})
			networkDefs[i].InterfaceRequest = podIfName
			mac, ok := macDetails[networkName]
			if ok {
				networkDefs[i].MacRequest = mac
			}
		}
	}

	networkDefByte, err := json.Marshal(networkDefs)
	if err != nil {
		return patchOps, err
	}

	pod.Annotations[networkv1.NetworkAttachmentAnnot] = string(networkDefByte)

	annotationBytes, err := json.Marshal(pod.Annotations)
	if err != nil {
		return patchOps, err
	}
	logrus.Debugf("multus annotation patch: %s\n", networkDefByte)
	fmt.Println(pod.Annotations)
	//"k8s.v1.cni.cncf.io/networks": "[{\"interface\":\"net1\",\"name\":\"workload\",\"namespace\":\"default\"}]",
	annotationPath := fmt.Sprintf("/metadata/annotations")

	containerBytes, err := generateHelperPatch(aliasRequest)
	if err != nil {
		return patchOps, err
	}
	containerPath := fmt.Sprintf("/spec/containers/-")
	patchOps = append(patchOps, fmt.Sprintf(`{"op": "replace", "path": "%s", "value": %s}`, annotationPath, string(annotationBytes)))
	patchOps = append(patchOps, fmt.Sprintf(`{"op": "add", "path": "%s", "value": %s}`, containerPath, string(containerBytes)))
	return patchOps, nil
}

// generateNetworkMacMap parses vmi status to generate a map
// of the form multus NAD definition name to associated MAC address
// eg: "default/workload": "96:c4:41:35:f9:2d"
func generateNetworkMacMap(vmi *kubevirtv1.VirtualMachineInstance) map[string]string {
	networkMacMapping := make(map[string]string)
	for _, v := range vmi.Status.Interfaces {
		networkMacMapping[v.Name] = v.MAC
	}
	result := make(map[string]string)
	for _, v := range vmi.Spec.Networks {
		if v.Multus != nil {
			result[v.Multus.NetworkName] = networkMacMapping[v.Name]
		}
	}
	return result
}

func generateHelperPatch(aliasRequest []networkhelperapi.NetworkMappingRequest) ([]byte, error) {
	args, err := json.Marshal(aliasRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshalling mapping request to json: %v", err)
	}
	container := corev1.Container{
		Name:            "harvester-network-migration-helper",
		Image:           "gmehta3/harvester-network-migration-helper:dev",
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"NET_ADMIN",
				},
			},
		},
		Args: []string{
			"--network-mapping-request",
			string(args),
		},
	}

	return json.Marshal(container)
}
