package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/terra-farm/go-virtualbox"
)

var (
	host     = flag.String("host", "0.0.0.0", "The interface for the webserver to listen on")
	port     = flag.String("port", "8080", "The port for the webserver to listen on")
	certFile = flag.String("cert-file", "./certs/cert.pem", "The path to the webserver TLS certificate")
	keyFile  = flag.String("key-file", "./certs/key.pem", "The path the the webserver TLS private key")
	token    = flag.String("token", "", "An organization level GitHub API token used to generate a Runner token inside of the VM")
)

type VirtualBoxServer struct {
	router *gin.Engine
}

type property struct {
	key   string
	value string
}

func (vbx *VirtualBoxServer) registerCreate() {
	vbx.router.GET("/create/:id/:owner/:repo/:image", func(context *gin.Context) {
		id := context.Param("id")
		owner := context.Param("owner")
		repo := context.Param("repo")
		image := context.Param("image")
		if id == "" || owner == "" || repo == "" || image == "" {
			context.JSON(http.StatusBadRequest, gin.H{
				"errMsg": "No VM ID present in the api URL params",
			})
			return
		}
		if id == "" {
			context.JSON(http.StatusBadRequest, gin.H{
				"errMsg": "No VM ID present in the api URL params",
			})
			return
		}
		err := virtualbox.CloneMachine(image, id, true)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{
				"errMsg": err.Error(),
			})
			return
		}
		vm, err := virtualbox.GetMachine(id)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{
				"errMsg": err.Error(),
			})
			return
		}
		err = vm.Start()
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{
				"errMsg": err.Error(),
			})
			return
		}
		props := []property{
			{"label", id},
			{"owner", owner},
			{"repo", repo},
			{"token", *token},
		}
		err = setGuestProperties(id, props...)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{
				"errMsg": err.Error(),
			})
			return
		}
		context.String(http.StatusOK, "Success")
	})
}

func (vbx *VirtualBoxServer) registerDelete() {
	vbx.router.GET("/delete/:id", func(context *gin.Context) {
		id := context.Param("id")
		if id == "" {
			context.JSON(300, gin.H{
				"errMsg": "No VM ID present in the api URL params",
			})
			return
		}
		vm, err := virtualbox.GetMachine(id)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{
				"errMsg": err.Error(),
			})
			return
		}
		err = vm.Delete()
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{
				"errMsg": err.Error(),
			})
			return
		}
		context.String(http.StatusOK, "Success")
	})
}

func (vbx *VirtualBoxServer) start() {
	address := fmt.Sprintf("%s:%s", *host, *port)
	panic(vbx.router.RunTLS(address, *certFile, *keyFile))
}

func setGuestProperties(vm string, props ...property) error {
	for _, prop := range props {
		err := virtualbox.SetGuestProperty(vm, prop.key, prop.value)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	flag.Parse()

	server := VirtualBoxServer{
		router: gin.Default(),
	}
	server.registerCreate()
	server.registerDelete()
	server.start()
}
