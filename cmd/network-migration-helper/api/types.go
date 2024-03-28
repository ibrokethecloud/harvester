package api

type NetworkMappingRequest struct {
	SourceInterface string `json:"sourceInterface"`
	AliasName       string `json:"aliasName"`
}
