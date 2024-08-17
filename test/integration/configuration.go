package integration

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)
import "github.com/docker/docker/api/types/network"

const proxyDockerfile = `
FROM golang:alpine as builder

WORKDIR /SpoofDPI

COPY . .

RUN go build -o spoof-dpi ./cmd/spoof-dpi/main.go

FROM alpine:latest

WORKDIR /

COPY --from=builder /SpoofDPI/spoof-dpi .
`

type TarConsumer interface {
	Accept(t *tar.Writer) error
}

type TarConsumerFn func(*tar.Writer) error

func (f TarConsumerFn) Accept(t *tar.Writer) error {
	return f(t)
}

type FilePredicate interface {
	Test(info FileInfo) bool
}

type FileInfo struct {
	AbsPath string
	Info    fs.FileInfo
}

type FilePredicateFn func(info FileInfo) bool

func (f FilePredicateFn) Test(info FileInfo) bool {
	return f(info)
}

func WriteHeaderAndContent(t *tar.Writer, h *tar.Header, content []byte) error {
	if err := t.WriteHeader(h); err != nil {
		return err
	}
	if _, err := t.Write(content); err != nil {
		return err
	}
	return nil
}

func WriteFile(t *tar.Writer, mode int64, name string, size int64, file io.Reader) error {
	hdr := &tar.Header{
		Name:     name,
		Mode:     mode,
		Size:     size,
		Typeflag: tar.TypeReg,
		Format:   tar.FormatGNU,
	}
	if content, err := io.ReadAll(file); err == nil {
		if err := WriteHeaderAndContent(t, hdr, content); err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}

func WriteDir(t *tar.Writer, mode int64, dirName string) error {
	return WriteHeaderAndContent(t, &tar.Header{
		Mode: int64(os.FileMode(mode) | os.ModeDir),
		Name: dirName,
		Size: 0,
	}, nil)
}

func MakeTar(sourceDir string, filePredicate FilePredicate, consumers ...TarConsumer) (io.Reader, error) {
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, err
	}

	filechan := make(chan FileInfo)
	walkCtx, cancelWalk := context.WithCancel(context.Background())
	defer cancelWalk()
	go func() {
		filepath.Walk(sourceDir, func(path string, info fs.FileInfo, err error) error {
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			select {
			case filechan <- FileInfo{
				Info:    info,
				AbsPath: abs,
			}:
			case <-walkCtx.Done():
				return fmt.Errorf("canceled")
			}
			return nil
		})
		close(filechan)
	}()

	var buf bytes.Buffer

	t := tar.NewWriter(&buf)

	for f := range filechan {
		if !(filePredicate.Test(f)) {
			continue
		}
		info := f.Info
		absFilePath := f.AbsPath
		relName, err := filepath.Rel(absSourceDir, absFilePath)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			if err := WriteDir(t, int64(info.Mode()), relName); err != nil {
				return nil, err
			}
		} else {
			file, err := os.OpenFile(absFilePath, os.O_RDONLY, 0644)
			if err != nil {
				return nil, err
			}
			if err = WriteFile(t, int64(info.Mode()), relName, info.Size(), file); err != nil {
				return nil, err
			}
		}
	}
	for _, consumer := range consumers {
		err := consumer.Accept(t)
		if err != nil {
			return nil, err
		}
	}
	if err := t.Close(); err != nil {
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
}

type Logging interface {
	Printf(format string, v ...interface{})
}

func SpoofDPIContainer(port uint16, log Logging, proxyRunArgs []string) (testcontainers.Container, error) {
	ctx := context.Background()
	projectTar, err := MakeTar("../../", FilePredicateFn(func(i FileInfo) bool {
		if i.Info.IsDir() {
			return false
		}
		if strings.Contains(i.AbsPath, ".git") {
			return false
		}
		if strings.Contains(i.AbsPath, "Dockerfile") {
			return false
		}
		if strings.Contains(i.AbsPath, ".idea") {
			return false
		}
		return true
	}), TarConsumerFn(func(writer *tar.Writer) error {
		return WriteFile(writer, 0644, "Dockerfile", int64(len(proxyDockerfile)), strings.NewReader(proxyDockerfile))
	}))
	if err != nil {
		return nil, err
	}

	cmd := []string{"./spoof-dpi"}
	cmd = append(cmd, "-port", strconv.Itoa(int(port)))
	cmd = append(cmd, proxyRunArgs...)
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{

		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Tag:            "spoof-dpi",
				Repo:           "spoof-dpi",
				PrintBuildLog:  true,
				ContextArchive: projectTar,
				Dockerfile:     "Dockerfile",
				KeepImage:      true,
			},
			WaitingFor: wait.ForLog("[PROXY]").WithOccurrence(1),
			Cmd:        cmd,
			LogConsumerCfg: &testcontainers.LogConsumerConfig{
				Consumers: []testcontainers.LogConsumer{
					&testcontainers.StdoutLogConsumer{},
				},
			},
			HostConfigModifier: func(config *container.HostConfig) {
				config.NetworkMode = network.NetworkHost
			},
		},
		Logger: log,
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}
