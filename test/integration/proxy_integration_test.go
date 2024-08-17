package integration_test

import (
	"context"
	"fmt"
	"io"
	rand2 "math/rand/v2"
	"net/http"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/test/integration"
)

type StdoutLogger struct {
}

func (*StdoutLogger) Printf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format, args...)
}

func TestTest(t *testing.T) {
	port := uint16(rand2.IntN(65536-2000) + 2000)
	container, err := integration.SpoofDPIContainer(port, new(StdoutLogger), []string{"-debug"})
	if err != nil {
		t.Fatal(err)
	}
	err = container.Start(context.Background())
	defer container.Terminate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	proxyHost := fmt.Sprintf("localhost:%d", port)
	logrus.Info("Started Proxy: ", proxyHost)
	if err != nil {
		t.Fatal(err)
	}
	os.Setenv("HTTPS_PROXY", proxyHost)
	os.Setenv("HTTP_PROXY", proxyHost)
	client := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	logrus.Info("resp:", string(body))
}
