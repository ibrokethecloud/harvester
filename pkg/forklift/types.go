package forklift

const (
	NetworkParentObjectKind = "Network"
)

type Datastore struct {
	ID                 string       `json:"id"`
	Parent             ParentObject `json:"parent"`
	Path               string       `json:"path"`
	Revision           int64        `json:"revision"`
	Name               string       `json:"name"`
	SelfLink           string       `json:"selfLink"`
	Type               string       `json:"type"`
	Capacity           int64        `json:"capacity"`
	Free               int64        `json:"free"`
	Maintenance        string       `json:"maintenance"`
	BackingDeviceNames []string     `json:"backingDeviceNames"`
}

type ParentObject struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type Network struct {
	ID       string       `json:"id"`
	Variant  string       `json:"variant"`
	Parent   ParentObject `json:"parent"`
	Path     string       `json:"path"`
	Revision int64        `json:"revision"`
	Name     string       `json:"name"`
	SelfLink string       `json:"selfLink"`
}

type VMNetworkAttachment struct {
	Network    ParentObject `json:"network"`
	MacAddress string       `json:"mac"`
	Order      int          `json:"order"`
}

type VMDiskAttachment struct {
	Key                   int32        `json:"key"`
	UnitNumber            int32        `json:"unitNumber"`
	ControllerKey         int32        `json:"controllerKey"`
	File                  string       `json:"file"`
	Datastore             ParentObject `json:"datastore"`
	Capacity              int64        `json:"capacity"`
	Shared                bool         `json:"shared"`
	RawDeviceMapping      bool         `json:"rdm"`
	Bus                   string       `json:"bus"`
	Mode                  string       `json:"mode"`
	Serial                string       `json:"serial"`
	ChangeTrackingenabled bool         `json:"changeTrackingEnabled"`
}

type VMConcern struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Category   string `json:"category"`
	Assessment string `json:"assessment"`
}

type GuestNetwork struct {
	Device     string   `json:"device"`
	MacAddress string   `json:"mac"`
	IPAddress  string   `json:"ip"`
	Origin     string   `json:"origin"`
	Prefix     int32    `json:"prefix"`
	DNS        []string `json:"dns"`
}

type GuestDisk struct {
	DiskPath       string `json:"diskPath"`
	Capacity       int64  `json:"capacity"`
	FreeSpace      int64  `json:"freeSpace"`
	FileSystemType string `json:"fileSystemType"`
}

type Device struct {
	Kind string `json:"kind"`
}

type VirtualMachines struct {
	ID                       string                `json:"id"`
	Parent                   ParentObject          `json:"parent"`
	Path                     string                `json:"path"`
	Revision                 int64                 `json:"revision"`
	Name                     string                `json:"name"`
	SelfLink                 string                `json:"selfLink"`
	LastValidatedRevision    string                `json:"lastValidatedRevision"`
	IsTemplate               bool                  `json:"isTemplate"`
	PowerState               string                `json:"powerState"`
	Host                     string                `json:"host"`
	Networks                 []ParentObject        `json:"networks"`
	Disks                    []VMDiskAttachment    `json:"disks"`
	Concerns                 []VMConcern           `json:"concerns"`
	PolicyVersion            int                   `json:"policyVersion"`
	UUID                     string                `json:"uuid"`
	Firmware                 string                `json:"firmware"`
	ConnectionState          string                `json:"connectionState"`
	Snapshot                 ParentObject          `json:"snapshot"`
	ChangeTrackingEnabled    bool                  `json:"changeTrackingEnabled"`
	CPUAffinity              []string              `json:"cpuAffinity"`
	CPUHotAddEnabled         bool                  `json:"cpuHotAddEnabled"`
	CPUHotRemoveEnabled      bool                  `json:"cpuHotRemoveEnabled"`
	MemoryHotAddEnabled      bool                  `json:"memoryHotAddEnabled"`
	FaultToleranceEnabled    bool                  `json:"faultToleranceEnabled"`
	CPUCount                 int32                 `json:"cpuCount"`
	MemoryMB                 int64                 `json:"memoryMB"`
	GuestName                string                `json:"guestName"`
	GuestNameFromVMWareTools string                `json:"guestNameFromVMWareTools"`
	Hostname                 string                `json:"hostname"`
	GuestID                  string                `json:"guestID"`
	BalloonedMemory          int64                 `json:"balloonedMemory"`
	IPAddress                string                `json:"ipAddress"`
	StorageUsed              int64                 `json:"storageUsed"`
	TPMEnabled               bool                  `json:"tpmEnabled"`
	NumaNodeAffinity         []string              `json:"numaNodeAffinity"`
	Devices                  []Device              `json:"devices"`
	NICS                     []VMNetworkAttachment `json:"nics"`
	GuestNetworks            []GuestNetwork        `json:"guestNetworks"`
	GuestDisks               []GuestDisk           `json:"guestDisks"`
	GuestIPStacks            []string              `json:"guestIPStacks"`
	SecureBoot               bool                  `json:"secureBoot"`
	DiskabeEnabledUUID       bool                  `json:"diskabeEnabledUUID"`
	NestedHVEnabled          bool                  `json:"nestedHVEnabled"`
}
