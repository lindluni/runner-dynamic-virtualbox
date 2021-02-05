package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

const ActionCreate = "create"
const ActionStart = "start"
const ActionDelete = "delete"

type VirtualBoxClient struct {
	gitClient  *github.Client
	httpClient *http.Client

	action string
	host   string
	image  string
	id     string
	owner  string
	port   string
	repo   string
	token  string
	cert   string
}

func main() {
	client := VirtualBoxClient{
		host:   os.Getenv("INPUT_HOST"),
		port:   os.Getenv("INPUT_PORT"),
		action: os.Getenv("INPUT_ACTION"),
		image:  os.Getenv("INPUT_IMAGE"),
		id:     os.Getenv("INPUT_PREFIX") + os.Getenv("INPUT_ID"),
		token:  os.Getenv("INPUT_TOKEN"),
		cert:   os.Getenv("INPUT_CERT"),
	}
	repo := os.Getenv("INPUT_REPO")
	client.owner = strings.Split(repo, "/")[0]
	client.repo = strings.Split(repo, "/")[1]

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: client.token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client.gitClient = github.NewClient(tc)
	client.httpClient = buildHTTPClient(client.cert)

	switch client.action {
	case ActionCreate:
		client.createVM()
		client.waitForRunner()
	case ActionDelete:
		client.deleteVM()
		client.deleteRunner()
	default:
		log.Panicf("Invalid action: %s", client.action)
	}

}

func (vbx *VirtualBoxClient) waitForRunner() {
	ctx := context.Background()
	for attempts := 30; attempts > 0; attempts-- {
		runners, _, err := vbx.gitClient.Actions.ListRunners(ctx, vbx.owner, vbx.repo, &github.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, runner := range runners.Runners {
			if vbx.id == runner.GetName() && runner.GetStatus() == "online" {
				return
			}
		}
		time.Sleep(10 * time.Second)
	}
	log.Panicf("Timed out waiting for runner %s to start", vbx.id)
}

func (vbx *VirtualBoxClient) createVM() {
	url := fmt.Sprintf("https://%s:%s/%s/%s/%s/%s/%s", vbx.host, vbx.port, ActionCreate, vbx.id, vbx.owner, vbx.repo, vbx.image)
	resp, err := vbx.httpClient.Get(url)
	if err != nil {
		log.Panicf("Create failed: Unable to reach endpoint: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panicf("Create failed: cannot parse response body: %v", err)
	}
	log.Panicf("Create failed: %s", string(bytes))
}

func (vbx *VirtualBoxClient) deleteVM() {
	url := fmt.Sprintf("https://%s:%s/%s/%s", vbx.host, vbx.port, ActionDelete, vbx.id)
	resp, err := vbx.httpClient.Get(url)
	if err != nil {
		log.Panicf("Delete failed: Unable to reach endpoint: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panicf("Delete failed: cannot parse response body: %v", err)
	}
	log.Panicf("Delete failed: %s", string(bytes))
}

func (vbx *VirtualBoxClient) deleteRunner() {
	ctx := context.Background()
	runners, _, err := vbx.gitClient.Actions.ListRunners(ctx, vbx.owner, vbx.repo, &github.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, runner := range runners.Runners {
		if vbx.id == runner.GetName() {
			_, err := vbx.gitClient.Actions.RemoveRunner(ctx, vbx.owner, vbx.repo, runner.GetID())
			if err != nil {
				log.Panicf("Delete failed: Unable to delete runner %s: %v", vbx.id, err)
			}
			return
		}
	}
	log.Panicf("Delete failed: Runner %s not found", vbx.id)
}

func buildHTTPClient(cert string) *http.Client {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		log.Panicf("Cannot build http.Client: Unable to retrieve system CertPool: %v", err)
	}
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	_ = rootCAs.AppendCertsFromPEM([]byte(cert))
	config := &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            rootCAs,
	}
	transport := &http.Transport{
		TLSClientConfig: config,
	}
	return &http.Client{
		Transport: transport,
	}
}
