package model

type Node struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	LastUpdated string       `json:"last_updated"`
	Devices     []DeviceInfo `json:"devices"`
}

type DeviceInfo struct {
	Serial   string `json:"serial"`
	Model    string `json:"model"`
	State    string `json:"state"`
	Product  string `json:"product"`
	Platform string `json:"platform"`
}
