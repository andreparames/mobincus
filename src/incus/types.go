package incus

type DockerContainer struct {
	ID        string   `json:"ID"`
	Names     []string `json:"Names"`
	Status    string   `json:"Status"`
	Image     string   `json:"Image"`
	Command   string   `json:"Command"`
	CreatedAt string   `json:"CreatedAt"`
	Ports     string   `json:"Ports"`
}
