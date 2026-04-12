//go:build darwin

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
	stateFilePath           = "/tmp/spoofdpi.proxy.darwin.prev"
	PermissionErrorHelpText = "By default spoofdpi tries to set itself up as a system-wide proxy server.\n" +
		"Doing so may require root access on machines with\n" +
		"'Settings > Privacy & Security > Advanced > Require" +
		" an administrator password to access system-wide settings' enabled.\n" +
		"If you do not want spoofdpi to act as a system-wide proxy, provide" +
		" -system-proxy=false."
)

type proxyStateDarwin struct {
	Service           string `json:"service"`
	PrevAutoProxyURL  string `json:"prev_autoproxy_url"`
	PrevAutoDiscovery string `json:"prev_autodiscovery"`
}

func RunPACServer(content string) (string, *http.Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
	})

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		_ = server.Serve(listener)
	}()

	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://127.0.0.1:%d/proxy.pac", addr.Port)

	return url, server, nil
}

func GetDefaultNetworkService() (string, error) {
	const cmd = "networksetup -listnetworkserviceorder | grep" +
		" `(route -n get default | grep 'interface' || route -n get -inet6 default | grep 'interface') | cut -d ':' -f2`" +
		" -B 1 | head -n 1 | cut -d ' ' -f 2-"

	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}

	service := strings.TrimSpace(string(out))
	if service == "" {
		return "", errors.New("no available networks")
	}
	return service, nil
}

func GetDefaultGateway() (string, error) {
	cmd := "route -n get default | grep gateway | awk '{print $2}'"
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func NetworkSetup(args ...string) error {
	cmd := exec.Command("networksetup", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if IsPermissionError(err) {
			msg += PermissionErrorHelpText
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func IsPermissionError(err error) bool {
	var exitErr *exec.ExitError
	ok := errors.As(err, &exitErr)
	return ok && exitErr.ExitCode() == 14
}

func getCurrentProxySettings(service string) (url, status string, err error) {
	cmd := exec.Command("networksetup", "-getautoproxyurl", service)
	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "URL:") {
		url = strings.TrimSpace(strings.TrimPrefix(lines[0], "URL:"))
		if url == "(null)" {
			url = ""
		}
	}

	cmd = exec.Command("networksetup", "-getproxyautodiscovery", service)
	out, err = cmd.Output()
	if err != nil {
		return "", "", err
	}

	output := strings.TrimSpace(string(out))
	if strings.HasSuffix(output, "Yes") {
		status = "on"
	} else {
		status = "off"
	}

	return url, status, nil
}

func saveState(state *proxyStateDarwin) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	if err := os.WriteFile(stateFilePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	return nil
}

func loadState() (*proxyStateDarwin, error) {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state proxyStateDarwin
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	return &state, nil
}

func deleteState() error {
	if err := os.Remove(stateFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}
	return nil
}

func SetSystemProxy(
	service string,
	port uint16,
	proxyType string,
) (*http.Server, error) {
	prevState, err := loadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	if prevState == nil {
		autoproxyURL, autoDiscovery, getErr := getCurrentProxySettings(service)
		if getErr != nil {
			return nil, getErr
		}
		prevState = &proxyStateDarwin{
			Service:           service,
			PrevAutoProxyURL:  autoproxyURL,
			PrevAutoDiscovery: autoDiscovery,
		}
	}

	if proxyType == "" {
		proxyType = "HTTP"
	}

	pacContent := fmt.Sprintf(`function FindProxyForURL(url, host) {
    return "%s 127.0.0.1:%d; DIRECT";
}`, proxyType, port)

	pacURL, pacServer, err := RunPACServer(pacContent)
	if err != nil {
		return nil, fmt.Errorf("error creating pac server: %w", err)
	}

	if err := NetworkSetup("-setautoproxyurl", service, pacURL); err != nil {
		_ = pacServer.Close()
		return nil, fmt.Errorf("setting autoproxyurl: %w", err)
	}

	if err := NetworkSetup("-setproxyautodiscovery", service, "on"); err != nil {
		_ = pacServer.Close()
		return nil, fmt.Errorf("setting proxyautodiscovery: %w", err)
	}

	if err := saveState(prevState); err != nil {
		_ = pacServer.Close()
		return nil, fmt.Errorf("failed to save state: %w", err)
	}

	return pacServer, nil
}

func UnsetSystemProxy(pacServer *http.Server) error {
	if pacServer != nil {
		_ = pacServer.Close()
	}

	state, err := loadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	if state == nil {
		return nil
	}

	var errs []error

	if state.PrevAutoProxyURL != "" {
		if err := NetworkSetup(
			"-setautoproxyurl",
			state.Service,
			state.PrevAutoProxyURL,
		); err != nil {
			errs = append(errs, err)
		}
	} else {
		if err := NetworkSetup("-setautoproxystate", state.Service, "off"); err != nil {
			errs = append(errs, err)
		}
	}

	if state.PrevAutoDiscovery == "on" {
		if err := NetworkSetup("-setproxyautodiscovery", state.Service, "on"); err != nil {
			errs = append(errs, err)
		}
	} else {
		if err := NetworkSetup("-setproxyautodiscovery", state.Service, "off"); err != nil {
			errs = append(errs, err)
		}
	}

	_ = deleteState()

	if len(errs) > 0 {
		return fmt.Errorf("failed to restore proxy settings: %v", errs)
	}
	return nil
}
