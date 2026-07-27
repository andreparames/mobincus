package incus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	Version    = "0.1.0"
	APIVersion = "1.0"

	DefaultUnixSocket = "/var/lib/incus/unix.socket"
	DefaultBaseURL    = "http://localhost/1.0"
)

type Client struct {
	BaseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				Dial: func(_, _ string) (net.Conn, error) {
					return net.DialTimeout("unix", DefaultUnixSocket, 5*time.Second)
				},
			},
		},
	}
}

type apiResponse struct {
	Type       string          `json:"type"`
	StatusCode int             `json:"status_code"`
	Status     string          `json:"status"`
	Error      string          `json:"error"`
	Operation  string          `json:"operation"`
	Metadata   json.RawMessage `json:"metadata"`
}

func (c *Client) request(method, path string, body interface{}) (*apiResponse, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, path)

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if apiResp.Type == "error" {
		return nil, fmt.Errorf("API error: %s", apiResp.Error)
	}

	return &apiResp, nil
}

func (c *Client) get(path string) (json.RawMessage, error) {
	path = strings.TrimPrefix(path, "/1.0")
	resp, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if resp.Type != "sync" {
		return nil, fmt.Errorf("unexpected response type: %s", resp.Type)
	}
	return resp.Metadata, nil
}

func (c *Client) post(path string, body interface{}) (*apiResponse, error) {
	return c.request("POST", path, body)
}

func (c *Client) put(path string, body interface{}) (*apiResponse, error) {
	return c.request("PUT", path, body)
}

func (c *Client) delete(path string) (*apiResponse, error) {
	return c.request("DELETE", path, nil)
}

type Operation struct {
	ID         string                 `json:"id"`
	Class      string                 `json:"class"`
	Status     string                 `json:"status"`
	StatusCode int                    `json:"status_code"`
	Metadata   map[string]interface{} `json:"metadata"`
	Err        string                 `json:"err"`
	Location   string                 `json:"location"`
}

func (c *Client) WaitOperation(opURL string) (*Operation, error) {
	path := strings.TrimPrefix(opURL, "/1.0")
	body, err := c.request("GET", path+"/wait", nil)
	if err != nil {
		return nil, err
	}
	var op Operation
	if err := json.Unmarshal(body.Metadata, &op); err != nil {
		return nil, fmt.Errorf("parsing operation: %w", err)
	}
	if op.Err != "" {
		return nil, fmt.Errorf("operation failed: %s", op.Err)
	}
	return &op, nil
}

type ServerEnvironment struct {
	Server         string `json:"server"`
	ServerVersion  string `json:"server_version"`
	ServerName     string `json:"server_name"`
	OSName         string `json:"os_name"`
	OSVersion      string `json:"os_version"`
	Kernel         string `json:"kernel"`
	KernelVersion  string `json:"kernel_version"`
	Driver         string `json:"driver"`
	DriverVersion  string `json:"driver_version"`
	Firewall       string `json:"firewall"`
	Storage        string `json:"storage"`
	StorageVersion string `json:"storage_version"`
}

type ServerInfo struct {
	APIVersion    string            `json:"api_version"`
	ServerVersion string            `json:"server_version"`
	Auth          string            `json:"auth"`
	AuthMethods   []string          `json:"auth_methods"`
	APIExtensions []string          `json:"api_extensions"`
	APIStatus     string            `json:"api_status"`
	Environment   ServerEnvironment `json:"environment"`
}

func (c *Client) GetServerInfo() (*ServerInfo, error) {
	metadata, err := c.get("")
	if err != nil {
		return nil, err
	}
	var info ServerInfo
	if err := json.Unmarshal(metadata, &info); err != nil {
		return nil, fmt.Errorf("parsing server info: %w", err)
	}
	return &info, nil
}

type Instance struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	StatusCode   int               `json:"status_code"`
	Type         string            `json:"type"`
	Architecture string            `json:"architecture"`
	Config       map[string]string `json:"config"`
	CreatedAt    string            `json:"created_at"`
	Description  string            `json:"description"`
	Ephemeral    bool              `json:"ephemeral"`
	Profiles     []string          `json:"profiles"`
	Project      string            `json:"project"`
}

func (c *Client) GetInstance(name string) (*Instance, error) {
	metadata, err := c.get("/instances/" + name)
	if err != nil {
		return nil, err
	}
	var detail Instance
	if err := json.Unmarshal(metadata, &detail); err != nil {
		return nil, fmt.Errorf("parsing instance detail: %w", err)
	}
	return &detail, nil
}

type InstanceSource struct {
	Type        string `json:"type"`
	Alias       string `json:"alias,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Server      string `json:"server,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
}

type InstanceCreateRequest struct {
	Name       string                 `json:"name"`
	Source     InstanceSource         `json:"source"`
	Ephemeral  bool                   `json:"ephemeral"`
	Config     map[string]string      `json:"config,omitempty"`
	Profiles   []string               `json:"profiles,omitempty"`
	Devices    map[string]interface{} `json:"devices,omitempty"`
}

type DiskDevice struct {
	Type   string `json:"type"`
	Pool   string `json:"pool"`
	Source string `json:"source"`
	Path   string `json:"path"`
}

type InstanceStatePut struct {
	Action  string `json:"action"`
	Timeout int    `json:"timeout,omitempty"`
}

type ExecPost struct {
	Command        []string          `json:"command"`
	Interactive    bool              `json:"interactive"`
	WaitForWS      bool              `json:"wait-for-websocket"`
	Environment    map[string]string `json:"environment,omitempty"`
	User           int               `json:"user,omitempty"`
	Group          int               `json:"group,omitempty"`
	Cwd            string            `json:"cwd,omitempty"`
	RecordOutput   bool              `json:"record-output,omitempty"`
}

func (c *Client) CreateInstance(req InstanceCreateRequest) (*Operation, error) {
	resp, err := c.post("/instances", req)
	if err != nil {
		return nil, err
	}
	if resp.Type == "sync" {
		return &Operation{Status: "Success"}, nil
	}
	return c.WaitOperation(resp.Operation)
}

func (c *Client) StartInstance(name string) error {
	resp, err := c.put("/instances/"+name+"/state", InstanceStatePut{Action: "start"})
	if err != nil {
		return err
	}
	if resp.Type == "async" {
		_, err = c.WaitOperation(resp.Operation)
	}
	return err
}

func (c *Client) StopInstance(name string, force bool) error {
	req := InstanceStatePut{Action: "stop"}
	if force {
		req.Timeout = 0
	}
	resp, err := c.put("/instances/"+name+"/state", req)
	if err != nil {
		return err
	}
	if resp.Type == "async" {
		_, err = c.WaitOperation(resp.Operation)
	}
	return err
}

func (c *Client) DeleteInstance(name string) error {
	resp, err := c.delete("/instances/" + name)
	if err != nil {
		return err
	}
	if resp.Type == "async" {
		_, err = c.WaitOperation(resp.Operation)
	}
	return err
}

type CustomVolume struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ContentType string `json:"content_type"`
	Config      map[string]string `json:"config"`
	UsedBy      []string `json:"used_by"`
	Location    string `json:"location"`
	Project     string `json:"project"`
}

func (c *Client) CreateVolume(name string) error {
	resp, err := c.post("/storage-pools/default/volumes/custom", map[string]string{"name": name})
	if err != nil {
		return err
	}
	if resp.Type == "async" {
		_, err = c.WaitOperation(resp.Operation)
	}
	return err
}

func (c *Client) ListVolumes() ([]string, error) {
	raw, err := c.get("/storage-pools/default/volumes/custom")
	if err != nil {
		return nil, err
	}
	var urls []string
	if err := json.Unmarshal(raw, &urls); err != nil {
		return nil, fmt.Errorf("parsing volume list: %w", err)
	}
	return urls, nil
}

func (c *Client) DeleteVolume(name string) error {
	resp, err := c.delete("/storage-pools/default/volumes/custom/" + name)
	if err != nil {
		return err
	}
	if resp.Type == "async" {
		_, err = c.WaitOperation(resp.Operation)
	}
	return err
}

func (c *Client) ExecInstance(name string, req ExecPost) (*apiResponse, error) {
	return c.post("/instances/"+name+"/exec", req)
}

type Image struct {
	Fingerprint string            `json:"fingerprint"`
	Aliases     []ImageAlias      `json:"aliases"`
	Properties  map[string]string `json:"properties"`
}

type ImageAlias struct {
	Name string `json:"name"`
}

func (c *Client) FindImage(name string) (InstanceSource, error) {
	base := extractImageName(name)
	localFP, err := c.findLocalImage(base)
	if err != nil {
		return InstanceSource{}, err
	}
	if localFP != "" {
		return InstanceSource{Type: "image", Fingerprint: localFP}, nil
	}

	return InstanceSource{
		Type:     "image",
		Alias:    base,
		Server:   "https://images.linuxcontainers.org",
		Protocol: "simplestreams",
	}, nil
}

func extractImageName(ref string) string {
	idx := strings.LastIndex(ref, "/")
	if idx >= 0 {
		ref = ref[idx+1:]
	}
	idx = strings.Index(ref, ":")
	if idx >= 0 {
		ref = ref[:idx]
	}
	return ref
}

func (c *Client) findLocalImage(name string) (string, error) {
	metadata, err := c.get("/images")
	if err != nil {
		return "", err
	}

	var urls []string
	if err := json.Unmarshal(metadata, &urls); err != nil {
		return "", fmt.Errorf("parsing image list: %w", err)
	}

	for _, url := range urls {
		imgMeta, err := c.get(url)
		if err != nil {
			continue
		}
		var img Image
		if err := json.Unmarshal(imgMeta, &img); err != nil {
			continue
		}
		for _, a := range img.Aliases {
			if a.Name == name {
				return img.Fingerprint, nil
			}
		}
		if img.Properties != nil {
			if strings.EqualFold(img.Properties["os"], name) {
				return img.Fingerprint, nil
			}
			if img.Properties["release"] == name {
				return img.Fingerprint, nil
			}
		}
	}

	return "", nil
}

func (c *Client) ListContainers() ([]DockerContainer, error) {
	metadata, err := c.get("/instances")
	if err != nil {
		return nil, err
	}

	var urls []string
	if err := json.Unmarshal(metadata, &urls); err != nil {
		return nil, fmt.Errorf("parsing instance list: %w", err)
	}

	var containers []DockerContainer
	for _, url := range urls {
		name := extractName(url)
		detail, err := c.GetInstance(name)
		if err != nil {
			return nil, err
		}
		cfg := make(map[string]string)
		for k, v := range detail.Config {
			if strings.HasPrefix(k, "user.") {
				cfg[k] = v
			}
		}
		containers = append(containers, DockerContainer{
			ID:        detail.Name,
			Names:     []string{detail.Name},
			Status:    detail.Status,
			Image:     "",
			CreatedAt: detail.CreatedAt,
			Ports:     "",
			Command:   "",
			Config:    cfg,
		})
	}

	return containers, nil
}

type FileInfo struct {
	Type     string // "file" or "directory"
	Size     int64
	Mode     os.FileMode
	UID      int
	GID      int
	Modified time.Time
}

func (c *Client) FileStat(name, path string) (*FileInfo, error) {
	req, err := http.NewRequest("HEAD", fmt.Sprintf("%s/instances/%s/files?path=%s", c.BaseURL, name, url.QueryEscape(path)), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("file stat error: HTTP %d", resp.StatusCode)
	}

	info := &FileInfo{
		Type: resp.Header.Get("X-Incus-Type"),
	}
	info.Size, _ = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	modeStr := resp.Header.Get("X-Incus-Mode")
	if modeStr != "" {
		if m, err := strconv.ParseInt(modeStr, 8, 32); err == nil {
			info.Mode = os.FileMode(m)
		}
	}
	info.UID, _ = strconv.Atoi(resp.Header.Get("X-Incus-Uid"))
	info.GID, _ = strconv.Atoi(resp.Header.Get("X-Incus-Gid"))
	modStr := resp.Header.Get("X-Incus-Modified")
	if modStr != "" {
		info.Modified, _ = time.Parse("2006-01-02 15:04:05 -0700 MST", modStr)
	}
	return info, nil
}

func (c *Client) FileGet(name, path string) (io.ReadCloser, *FileInfo, error) {
	info, err := c.FileStat(name, path)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/instances/%s/files?path=%s", c.BaseURL, name, url.QueryEscape(path)), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("file get error: HTTP %d", resp.StatusCode)
	}
	return resp.Body, info, nil
}

type fileListResponse struct {
	Metadata []string `json:"metadata"`
}

func (c *Client) FileList(name, path string) ([]string, error) {
	raw, err := c.get("/instances/" + name + "/files?path=" + url.QueryEscape(path))
	if err != nil {
		return nil, err
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("parsing file list: %w", err)
	}
	return list, nil
}

func (c *Client) FilePut(name, path, fileType string, mode os.FileMode, content io.Reader) error {
	u := fmt.Sprintf("%s/instances/%s/files?path=%s", c.BaseURL, name, url.QueryEscape(path))
	req, err := http.NewRequest("POST", u, content)
	if err != nil {
		return err
	}
	req.Header.Set("X-Incus-Type", fileType)
	req.Header.Set("X-Incus-Mode", fmt.Sprintf("%04o", mode))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("file put error: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) SetConfig(name, key, value string) error {
	inst, err := c.GetInstance(name)
	if err != nil {
		return err
	}

	if inst.Config == nil {
		inst.Config = make(map[string]string)
	}
	inst.Config[key] = value

	resp, err := c.put("/instances/"+name, map[string]interface{}{
		"config":      inst.Config,
		"description": inst.Description,
		"ephemeral":   inst.Ephemeral,
		"profiles":    inst.Profiles,
		"devices":     map[string]interface{}{},
	})
	if err != nil {
		return err
	}
	if resp.Type == "async" {
		_, err = c.WaitOperation(resp.Operation)
	}
	return err
}

func (c *Client) GetConfig(name, key string) (string, error) {
	inst, err := c.GetInstance(name)
	if err != nil {
		return "", err
	}
	return inst.Config[key], nil
}

func (c *Client) WaitInstanceStopped(name string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		inst, err := c.GetInstance(name)
		if err != nil {
			return "", err
		}
		if inst.Status == "Stopped" {
			return c.GetConfig(name, "user.docker_exit_code")
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", fmt.Errorf("timed out waiting for instance %s to stop", name)
}

func extractName(url string) string {
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			return url[i+1:]
		}
	}
	return url
}
