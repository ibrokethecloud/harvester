package forklift

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
