package pod

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/harvester/harvester/pkg/util"
	"github.com/harvester/harvester/pkg/webhook/types"
)

func Test_envPatches(t *testing.T) {

	type input struct {
		targetEnvs []corev1.EnvVar
		proxyEnvs  []corev1.EnvVar
		basePath   string
	}
	var testCases = []struct {
		name   string
		input  input
		output types.PatchOps
	}{
		{
			name: "add proxy envs",
			input: input{
				targetEnvs: []corev1.EnvVar{
					{
						Name:  "foo",
						Value: "bar",
					},
				},
				proxyEnvs: []corev1.EnvVar{
					{
						Name:  util.HTTPProxyEnv,
						Value: "http://192.168.0.1:3128",
					},
					{
						Name:  util.HTTPSProxyEnv,
						Value: "http://192.168.0.1:3128",
					},
					{
						Name:  util.NoProxyEnv,
						Value: "127.0.0.1,0.0.0.0,10.0.0.0/8",
					},
				},
				basePath: "/spec/containers/0/env",
			},
			output: []string{
				`{"op": "add", "path": "/spec/containers/0/env/-", "value": {"name":"HTTP_PROXY","value":"http://192.168.0.1:3128"}}`,
				`{"op": "add", "path": "/spec/containers/0/env/-", "value": {"name":"HTTPS_PROXY","value":"http://192.168.0.1:3128"}}`,
				`{"op": "add", "path": "/spec/containers/0/env/-", "value": {"name":"NO_PROXY","value":"127.0.0.1,0.0.0.0,10.0.0.0/8"}}`,
			},
		},
		{
			name: "add proxy envs to empty envs",
			input: input{
				targetEnvs: []corev1.EnvVar{},
				proxyEnvs: []corev1.EnvVar{
					{
						Name:  util.HTTPProxyEnv,
						Value: "http://192.168.0.1:3128",
					},
					{
						Name:  util.HTTPSProxyEnv,
						Value: "http://192.168.0.1:3128",
					},
					{
						Name:  util.NoProxyEnv,
						Value: "127.0.0.1,0.0.0.0,10.0.0.0/8",
					},
				},
				basePath: "/spec/containers/0/env",
			},
			output: []string{
				`{"op": "add", "path": "/spec/containers/0/env", "value": [{"name":"HTTP_PROXY","value":"http://192.168.0.1:3128"}]}`,
				`{"op": "add", "path": "/spec/containers/0/env/-", "value": {"name":"HTTPS_PROXY","value":"http://192.168.0.1:3128"}}`,
				`{"op": "add", "path": "/spec/containers/0/env/-", "value": {"name":"NO_PROXY","value":"127.0.0.1,0.0.0.0,10.0.0.0/8"}}`,
			},
		},
	}
	for _, testCase := range testCases {
		result, err := envPatches(testCase.input.targetEnvs, testCase.input.proxyEnvs, testCase.input.basePath)
		assert.Equal(t, testCase.output, result)
		assert.Empty(t, err)
	}
}

func Test_volumePatch(t *testing.T) {

	type input struct {
		target []corev1.Volume
		volume corev1.Volume
	}
	var testCases = []struct {
		name   string
		input  input
		output string
	}{
		{
			name: "add additional ca volume",
			input: input{
				target: []corev1.Volume{
					{
						Name: "foo",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				volume: corev1.Volume{
					Name: "additional-ca-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: pointer.Int32(400),
							SecretName:  util.AdditionalCASecretName,
						},
					},
				},
			},
			output: `{"op": "add", "path": "/spec/volumes/-", "value": {"name":"additional-ca-volume","secret":{"secretName":"harvester-additional-ca","defaultMode":400}}}`,
		},
		{
			name: "add additional ca volume to empty volumes",
			input: input{
				target: []corev1.Volume{},
				volume: corev1.Volume{
					Name: "additional-ca-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: pointer.Int32(400),
							SecretName:  util.AdditionalCASecretName,
						},
					},
				},
			},
			output: `{"op": "add", "path": "/spec/volumes", "value": [{"name":"additional-ca-volume","secret":{"secretName":"harvester-additional-ca","defaultMode":400}}]}`,
		},
	}
	for _, testCase := range testCases {
		result, err := volumePatch(testCase.input.target, testCase.input.volume)
		assert.Equal(t, testCase.output, result)
		assert.Empty(t, err)
	}
}

func Test_volumeMountPatch(t *testing.T) {

	type input struct {
		target      []corev1.VolumeMount
		volumeMount corev1.VolumeMount
		path        string
	}
	var testCases = []struct {
		name   string
		input  input
		output string
	}{
		{
			name: "add additional ca volume mount",
			input: input{
				target: []corev1.VolumeMount{
					{
						Name:      "foo",
						MountPath: "/bar",
					},
				},
				volumeMount: corev1.VolumeMount{
					Name:      "additional-ca-volume",
					MountPath: "/etc/ssl/certs/" + util.AdditionalCAFileName,
					SubPath:   util.AdditionalCAFileName,
					ReadOnly:  true,
				},
				path: "/spec/containers/0/volumeMounts",
			},
			output: `{"op": "add", "path": "/spec/containers/0/volumeMounts/-", "value": {"name":"additional-ca-volume","readOnly":true,"mountPath":"/etc/ssl/certs/additional-ca.pem","subPath":"additional-ca.pem"}}`,
		},
		{
			name: "add additional ca volume mount to empty volumeMounts",
			input: input{
				target: []corev1.VolumeMount{},
				volumeMount: corev1.VolumeMount{
					Name:      "additional-ca-volume",
					MountPath: "/etc/ssl/certs/" + util.AdditionalCAFileName,
					SubPath:   util.AdditionalCAFileName,
					ReadOnly:  true,
				},
				path: "/spec/containers/0/volumeMounts",
			},
			output: `{"op": "add", "path": "/spec/containers/0/volumeMounts", "value": [{"name":"additional-ca-volume","readOnly":true,"mountPath":"/etc/ssl/certs/additional-ca.pem","subPath":"additional-ca.pem"}]}`,
		},
	}
	for _, testCase := range testCases {
		result, err := volumeMountPatch(testCase.input.target, testCase.input.path, testCase.input.volumeMount)
		assert.Equal(t, testCase.output, result)
		assert.Empty(t, err)
	}
}

func Test_generateMultusAnnotationPatch(t *testing.T) {
	podJSONBytes := []byte(`{
		"apiVersion": "v1",
		"kind": "Pod",
		"metadata": {
			"annotations": {
				"cni.projectcalico.org/containerID": "e175ee39d4caa7a159cd8ef2fc144ef4ddea380dd119ef1955614c86e086af1b",
				"cni.projectcalico.org/podIP": "10.52.2.16/32",
				"cni.projectcalico.org/podIPs": "10.52.2.16/32",
				"harvesterhci.io/sshNames": "[]",
				"k8s.v1.cni.cncf.io/networks": "[{\"interface\":\"net1\",\"name\":\"workload\",\"namespace\":\"default\"}]",
				"kubectl.kubernetes.io/default-container": "compute",
				"kubevirt.io/domain": "vm2",
				"kubevirt.io/migrationTransportUnix": "true",
				"post.hook.backup.velero.io/command": "[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", \"vm2\", \"--namespace\", \"default\"]",
				"post.hook.backup.velero.io/container": "compute",
				"pre.hook.backup.velero.io/command": "[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", \"vm2\", \"--namespace\", \"default\"]",
				"pre.hook.backup.velero.io/container": "compute"
			},
			"creationTimestamp": "2024-02-16T04:39:38Z",
			"generateName": "virt-launcher-vm2-",
			"labels": {
				"harvesterhci.io/vmName": "vm2",
				"kubevirt.io": "virt-launcher",
				"kubevirt.io/created-by": "39f1e42d-3f7f-44da-88cd-18f01b65f9be",
				"kubevirt.io/nodeName": "vm3",
				"kubevirt.io/outdatedLauncherImage": "",
				"vm.kubevirt.io/name": "vm2"
			},
			"name": "virt-launcher-vm2-mgvbp",
			"namespace": "default",
			"ownerReferences": [
				{
					"apiVersion": "kubevirt.io/v1",
					"blockOwnerDeletion": true,
					"controller": true,
					"kind": "VirtualMachineInstance",
					"name": "vm2",
					"uid": "39f1e42d-3f7f-44da-88cd-18f01b65f9be"
				}
			],
			"resourceVersion": "97475",
			"uid": "9732ac62-3046-45ce-bb92-fadfc8bc6601"
		},
		"spec": {
			"affinity": {
				"nodeAffinity": {
					"requiredDuringSchedulingIgnoredDuringExecution": {
						"nodeSelectorTerms": [
							{
								"matchExpressions": [
									{
										"key": "network.harvesterhci.io/mgmt",
										"operator": "In",
										"values": [
											"true"
										]
									}
								]
							}
						]
					}
				}
			},
			"automountServiceAccountToken": false,
			"containers": [
				{
					"command": [
						"/usr/bin/virt-launcher-monitor",
						"--qemu-timeout",
						"332s",
						"--name",
						"vm2",
						"--uid",
						"39f1e42d-3f7f-44da-88cd-18f01b65f9be",
						"--namespace",
						"default",
						"--kubevirt-share-dir",
						"/var/run/kubevirt",
						"--ephemeral-disk-dir",
						"/var/run/kubevirt-ephemeral-disks",
						"--container-disk-dir",
						"/var/run/kubevirt/container-disks",
						"--grace-period-seconds",
						"135",
						"--hook-sidecars",
						"0",
						"--ovmf-path",
						"/usr/share/OVMF"
					],
					"env": [
						{
							"name": "KUBEVIRT_RESOURCE_NAME_default"
						},
						{
							"name": "POD_NAME",
							"valueFrom": {
								"fieldRef": {
									"apiVersion": "v1",
									"fieldPath": "metadata.name"
								}
							}
						}
					],
					"image": "registry.suse.com/suse/sles/15.4/virt-launcher:0.54.0-150400.3.19.1",
					"imagePullPolicy": "IfNotPresent",
					"name": "compute",
					"resources": {
						"limits": {
							"cpu": "1",
							"devices.kubevirt.io/kvm": "1",
							"devices.kubevirt.io/tun": "1",
							"devices.kubevirt.io/vhost-net": "1",
							"memory": "2372577281"
						},
						"requests": {
							"cpu": "62m",
							"devices.kubevirt.io/kvm": "1",
							"devices.kubevirt.io/tun": "1",
							"devices.kubevirt.io/vhost-net": "1",
							"ephemeral-storage": "50M",
							"memory": "1656399873"
						}
					},
					"securityContext": {
						"capabilities": {
							"add": [
								"NET_BIND_SERVICE",
								"SYS_PTRACE",
								"SYS_NICE"
							],
							"drop": [
								"NET_RAW"
							]
						},
						"privileged": false,
						"runAsNonRoot": false,
						"runAsUser": 0
					},
					"terminationMessagePath": "/dev/termination-log",
					"terminationMessagePolicy": "File",
					"volumeDevices": [
						{
							"devicePath": "/dev/disk-0",
							"name": "disk-0"
						}
					],
					"volumeMounts": [
						{
							"mountPath": "/var/run/kubevirt-private",
							"name": "private"
						},
						{
							"mountPath": "/var/run/kubevirt",
							"name": "public"
						},
						{
							"mountPath": "/var/run/kubevirt-ephemeral-disks",
							"name": "ephemeral-disks"
						},
						{
							"mountPath": "/var/run/kubevirt/container-disks",
							"mountPropagation": "HostToContainer",
							"name": "container-disks"
						},
						{
							"mountPath": "/var/run/kubevirt/hotplug-disks",
							"mountPropagation": "HostToContainer",
							"name": "hotplug-disks"
						},
						{
							"mountPath": "/var/run/libvirt",
							"name": "libvirt-runtime"
						},
						{
							"mountPath": "/var/run/kubevirt/sockets",
							"name": "sockets"
						},
						{
							"mountPath": "/var/run/kubevirt-private/secret/cloudinitdisk/userdata",
							"name": "cloudinitdisk-udata",
							"readOnly": true,
							"subPath": "userdata"
						},
						{
							"mountPath": "/var/run/kubevirt-private/secret/cloudinitdisk/userData",
							"name": "cloudinitdisk-udata",
							"readOnly": true,
							"subPath": "userData"
						},
						{
							"mountPath": "/var/run/kubevirt-private/secret/cloudinitdisk/networkdata",
							"name": "cloudinitdisk-ndata",
							"readOnly": true,
							"subPath": "networkdata"
						},
						{
							"mountPath": "/var/run/kubevirt-private/secret/cloudinitdisk/networkData",
							"name": "cloudinitdisk-ndata",
							"readOnly": true,
							"subPath": "networkData"
						}
					]
				}
			],
			"dnsPolicy": "ClusterFirst",
			"enableServiceLinks": false,
			"hostname": "vm2",
			"nodeName": "vm3",
			"nodeSelector": {
				"kubevirt.io/schedulable": "true"
			},
			"preemptionPolicy": "PreemptLowerPriority",
			"priority": 0,
			"readinessGates": [
				{
					"conditionType": "kubevirt.io/virtual-machine-unpaused"
				}
			],
			"restartPolicy": "Never",
			"schedulerName": "default-scheduler",
			"securityContext": {
				"runAsUser": 0,
				"seLinuxOptions": {
					"type": "virt_launcher.process"
				}
			},
			"serviceAccount": "default",
			"serviceAccountName": "default",
			"terminationGracePeriodSeconds": 150,
			"tolerations": [
				{
					"effect": "NoExecute",
					"key": "node.kubernetes.io/not-ready",
					"operator": "Exists",
					"tolerationSeconds": 300
				},
				{
					"effect": "NoExecute",
					"key": "node.kubernetes.io/unreachable",
					"operator": "Exists",
					"tolerationSeconds": 300
				}
			],
			"volumes": [
				{
					"emptyDir": {},
					"name": "private"
				},
				{
					"emptyDir": {},
					"name": "public"
				},
				{
					"emptyDir": {},
					"name": "sockets"
				},
				{
					"name": "disk-0",
					"persistentVolumeClaim": {
						"claimName": "vm2-disk-0-ghxn9"
					}
				},
				{
					"name": "cloudinitdisk-udata",
					"secret": {
						"defaultMode": 420,
						"secretName": "vm2-ub3db"
					}
				},
				{
					"name": "cloudinitdisk-ndata",
					"secret": {
						"defaultMode": 420,
						"secretName": "vm2-ub3db"
					}
				},
				{
					"emptyDir": {},
					"name": "virt-bin-share-dir"
				},
				{
					"emptyDir": {},
					"name": "libvirt-runtime"
				},
				{
					"emptyDir": {},
					"name": "ephemeral-disks"
				},
				{
					"emptyDir": {},
					"name": "container-disks"
				},
				{
					"emptyDir": {},
					"name": "hotplug-disks"
				}
			]
		},
		"status": {
			"conditions": [
				{
					"lastProbeTime": "2024-02-16T04:39:38Z",
					"lastTransitionTime": "2024-02-16T04:39:38Z",
					"message": "the virtual machine is not paused",
					"reason": "NotPaused",
					"status": "True",
					"type": "kubevirt.io/virtual-machine-unpaused"
				},
				{
					"lastProbeTime": null,
					"lastTransitionTime": "2024-02-16T04:39:42Z",
					"status": "True",
					"type": "Initialized"
				},
				{
					"lastProbeTime": null,
					"lastTransitionTime": "2024-02-16T04:39:57Z",
					"status": "True",
					"type": "Ready"
				},
				{
					"lastProbeTime": null,
					"lastTransitionTime": "2024-02-16T04:39:57Z",
					"status": "True",
					"type": "ContainersReady"
				},
				{
					"lastProbeTime": null,
					"lastTransitionTime": "2024-02-16T04:39:42Z",
					"status": "True",
					"type": "PodScheduled"
				}
			],
			"containerStatuses": [
				{
					"containerID": "containerd://d97599ec7017af98894638dff9f15dbc39531db395a1a6eb275035cd8d70404b",
					"image": "registry.suse.com/suse/sles/15.4/virt-launcher:0.54.0-150400.3.19.1",
					"imageID": "sha256:9a88c4e760797185a774ed22bd71f1f5d0b07600df944b230cca8f93518e956a",
					"lastState": {},
					"name": "compute",
					"ready": true,
					"restartCount": 0,
					"started": true,
					"state": {
						"running": {
							"startedAt": "2024-02-16T04:39:57Z"
						}
					}
				}
			],
			"hostIP": "192.168.122.242",
			"phase": "Running",
			"podIP": "10.52.2.16",
			"podIPs": [
				{
					"ip": "10.52.2.16"
				}
			],
			"qosClass": "Burstable",
			"startTime": "2024-02-16T04:39:42Z"
		}
	}`)

	vmiJSONBytes := []byte(`{
		"apiVersion": "kubevirt.io/v1",
		"kind": "VirtualMachineInstance",
		"metadata": {
			"annotations": {
				"harvesterhci.io/sshNames": "[]",
				"kubevirt.io/latest-observed-api-version": "v1",
				"kubevirt.io/storage-observed-api-version": "v1alpha3",
				"kubevirt.io/vm-generation": "1"
			},
			"creationTimestamp": "2024-02-16T04:39:38Z",
			"finalizers": [
				"kubevirt.io/virtualMachineControllerFinalize",
				"foregroundDeleteVirtualMachine",
				"wrangler.cattle.io/harvester-lb-vmi-controller",
				"wrangler.cattle.io/VMIController.UnsetOwnerOfPVCs"
			],
			"generation": 15,
			"labels": {
				"harvesterhci.io/vmName": "vm2",
				"kubevirt.io/nodeName": "vm3",
				"kubevirt.io/outdatedLauncherImage": ""
			},
			"name": "vm2",
			"namespace": "default",
			"ownerReferences": [
				{
					"apiVersion": "kubevirt.io/v1",
					"blockOwnerDeletion": true,
					"controller": true,
					"kind": "VirtualMachine",
					"name": "vm2",
					"uid": "312e4b81-c903-4126-b792-5c020041f611"
				}
			],
			"resourceVersion": "97482",
			"uid": "39f1e42d-3f7f-44da-88cd-18f01b65f9be"
		},
		"spec": {
			"affinity": {
				"nodeAffinity": {
					"requiredDuringSchedulingIgnoredDuringExecution": {
						"nodeSelectorTerms": [
							{
								"matchExpressions": [
									{
										"key": "network.harvesterhci.io/mgmt",
										"operator": "In",
										"values": [
											"true"
										]
									}
								]
							}
						]
					}
				}
			},
			"domain": {
				"cpu": {
					"cores": 1,
					"model": "host-model",
					"sockets": 1,
					"threads": 1
				},
				"devices": {
					"disks": [
						{
							"bootOrder": 1,
							"disk": {
								"bus": "virtio"
							},
							"name": "disk-0"
						},
						{
							"disk": {
								"bus": "virtio"
							},
							"name": "cloudinitdisk"
						}
					],
					"inputs": [
						{
							"bus": "usb",
							"name": "tablet",
							"type": "tablet"
						}
					],
					"interfaces": [
						{
							"bridge": {},
							"model": "virtio",
							"name": "default"
						}
					]
				},
				"features": {
					"acpi": {
						"enabled": true
					}
				},
				"firmware": {
					"uuid": "a8f84d8e-eaaa-5622-97db-8ce003bc7c5f"
				},
				"machine": {
					"type": "q35"
				},
				"memory": {
					"guest": "1948Mi"
				},
				"resources": {
					"limits": {
						"cpu": "1",
						"memory": "2Gi"
					},
					"requests": {
						"cpu": "62m",
						"memory": "1365Mi"
					}
				}
			},
			"evictionStrategy": "LiveMigrate",
			"hostname": "vm2",
			"networks": [
				{
					"multus": {
						"networkName": "default/workload"
					},
					"name": "default"
				}
			],
			"terminationGracePeriodSeconds": 120,
			"volumes": [
				{
					"name": "disk-0",
					"persistentVolumeClaim": {
						"claimName": "vm2-disk-0-ghxn9"
					}
				},
				{
					"cloudInitNoCloud": {
						"networkDataSecretRef": {
							"name": "vm2-ub3db"
						},
						"secretRef": {
							"name": "vm2-ub3db"
						}
					},
					"name": "cloudinitdisk"
				}
			]
		},
		"status": {
			"activePods": {
				"9732ac62-3046-45ce-bb92-fadfc8bc6601": "vm3"
			},
			"conditions": [
				{
					"lastProbeTime": null,
					"lastTransitionTime": "2024-02-16T04:39:57Z",
					"status": "True",
					"type": "Ready"
				},
				{
					"lastProbeTime": null,
					"lastTransitionTime": null,
					"status": "True",
					"type": "LiveMigratable"
				},
				{
					"lastProbeTime": "2024-02-16T04:41:57Z",
					"lastTransitionTime": null,
					"status": "True",
					"type": "AgentConnected"
				}
			],
			"guestOSInfo": {
				"id": "ubuntu",
				"kernelRelease": "5.15.0-79-generic",
				"kernelVersion": "#86-Ubuntu SMP Mon Jul 10 16:07:21 UTC 2023",
				"name": "Ubuntu",
				"prettyName": "Ubuntu 22.04.3 LTS",
				"version": "22.04",
				"versionId": "22.04"
			},
			"interfaces": [
				{
					"infoSource": "domain, guest-agent, multus-status",
					"interfaceName": "enp1s0",
					"ipAddress": "192.168.122.85",
					"ipAddresses": [
						"192.168.122.85",
						"fe80::7c0e:80ff:fe20:19f5"
					],
					"mac": "7e:0e:80:20:19:f5",
					"name": "default",
					"queueCount": 1
				}
			],
			"launcherContainerImageVersion": "registry.suse.com/suse/sles/15.4/virt-launcher:0.54.0-150400.3.19.1",
			"machine": {
				"type": "pc-q35-6.2"
			},
			"migrationMethod": "BlockMigration",
			"migrationTransport": "Unix",
			"nodeName": "vm3",
			"phase": "Running",
			"phaseTransitionTimestamps": [
				{
					"phase": "Pending",
					"phaseTransitionTimestamp": "2024-02-16T04:39:38Z"
				},
				{
					"phase": "Scheduling",
					"phaseTransitionTimestamp": "2024-02-16T04:39:38Z"
				},
				{
					"phase": "Scheduled",
					"phaseTransitionTimestamp": "2024-02-16T04:39:57Z"
				},
				{
					"phase": "Running",
					"phaseTransitionTimestamp": "2024-02-16T04:40:01Z"
				}
			],
			"qosClass": "Burstable",
			"runtimeUser": 0,
			"selinuxContext": "none",
			"virtualMachineRevisionName": "revision-start-vm-312e4b81-c903-4126-b792-5c020041f611-1",
			"volumeStatus": [
				{
					"name": "cloudinitdisk",
					"size": 1048576,
					"target": "vdb"
				},
				{
					"name": "disk-0",
					"persistentVolumeClaimInfo": {
						"accessModes": [
							"ReadWriteMany"
						],
						"capacity": {
							"storage": "10Gi"
						},
						"filesystemOverhead": "0.055",
						"requests": {
							"storage": "10Gi"
						},
						"volumeMode": "Block"
					},
					"target": "vda"
				}
			]
		}
	}`)

	pod := &corev1.Pod{}
	vmi := &kubevirtv1.VirtualMachineInstance{}
	assert := require.New(t)
	err := json.Unmarshal(podJSONBytes, pod)
	assert.NoError(err)
	err = json.Unmarshal(vmiJSONBytes, vmi)
	assert.NoError(err)
	assert.True(IsKubevirtLauncherPod(pod))
	ok, vm := podOwnedByVMI(pod)
	assert.True(ok)
	assert.Equal(vm, vmi.Name)
	patchOps, err := generateMultusAnnotationPatch(vmi, pod)
	t.Log(patchOps)
	assert.NoError(err)
	assert.Len(patchOps, 1)
}
