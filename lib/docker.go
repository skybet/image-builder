// Copyright (c) 2017 Adam Pointer

package lib

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type buildResponse struct {
	Stream string `json:"stream"`
	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
}

type pushResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type DockerClient struct {
	cli *client.Client
}

func GetDockerClient(host string) (*DockerClient, error) {
	var c *client.Client

	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	c, err := client.NewClient(host, "v1.24", nil, defaultHeaders)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to Docker daemon on %s: %s", host, err)
	}
	return newDockerClient(c), nil
}

// NewDockerClient returns a new DockerClient initialised with the API object
func newDockerClient(cli *client.Client) *DockerClient {
	c := new(DockerClient)
	c.cli = cli
	return c
}

func (c *DockerClient) Build(buildCtx io.Reader, tags []string) (err error) {
	opts := types.ImageBuildOptions{
		Remove: true,
		Tags:   tags,
	}
	resp, err := c.cli.ImageBuild(context.Background(), buildCtx, opts)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		txt := scanner.Text()
		var r buildResponse
		if err := json.Unmarshal([]byte(txt), &r); err != nil {
			return fmt.Errorf("Error decoding json from build image API: %s", err)
		}
		if r.Error != "" {
			return fmt.Errorf("Build API: %s", r.Error)
		}
		log.Debug(strings.TrimSuffix(r.Stream, "\n"))
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Failed to read logs: %s", err)
	}
	return
}

func (c *DockerClient) Push(repo string, auth string) (err error) {
	opts := types.ImagePushOptions{
		All:          true,
		RegistryAuth: auth,
	}
	resp, err := c.cli.ImagePush(context.Background(), repo, opts)
	if err != nil {
		return err
	}
	defer resp.Close()

	scanner := bufio.NewScanner(resp)
	for scanner.Scan() {
		txt := scanner.Text()
		var r pushResponse
		if err := json.Unmarshal([]byte(txt), &r); err != nil {
			return fmt.Errorf("Error decoding json from push image API: %s", err)
		}
		if r.Error != "" {
			return fmt.Errorf("Push API: %s", r.Error)
		}
		log.WithFields(log.Fields{
			"layer": r.Id,
		}).Debug(r.Status)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Failed to read logs: %s", err)
	}
	return
}
