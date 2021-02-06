# VirtualBox Dynamic Provisioner

The VirtualBox Dynamic Provisioner (VBDP) is a client-server GitHub Action that allows users to
provision virtual machines using VirtualBox at workflow runtime. Users can control the
provisioning and destruction of vm's on any infrastructure where you can install VirtualBox.

This has major advantages over the GitHub hosted runners in that it allows you to bring any
custom hardware you have access to. If you require GPU-enabled vm's, or processors running
custom secure-enclave processors, such as Intel-SGX, users can make use of the VBDP to
enable their workflow to run on their custom hardware.

**An end-to-end installation example can be found [here](INSTALL.md)**

## Client-Server Model

The VirtualBox Dynamic Provisioner is made up of two components:

### VirtualBoxServer

The VirtualBoxServer is a Go-based TLS-enabled REST server that runs on a hardware host. The server 
communicates with a client that instructs it to provision or destroy vm's. The server receives API
requests along with metadata required to configure the server and creates or destroys vm's on demand
by interacting with the VirtualBoxManage API available on the host.

### VirtualBoxClient

The VirtualBoxClient is a Go-based client that interacts with a VirtualBoxServer. Unlike the server,
it manages not only submitting request to provision or destroy vm's on a host, but also manages
the removal of the registration of the runner from GitHub once the vm is destroyed.

While the client can be run from anywhere, it was designed first and foremost to be executed as 
a Docker-based GitHub Action.

## Overview

### Creating a virtual machine image

The VirtualBoxServer uses the clone functionality of VirtualBox to create new vm's on demand. Users can
create as many vm's as they need and make use of these images in their jobs. This makes the Action
highly flexible.

This repository contains the necessary utility scripts to bootstrap a Linux server in the scripts directory:

`setup.sh` - This script is used when building a Linux-based image to create the necessary directories,
install the necessary dependencies, and place the required runner setup files in the vm image. The script
will create the runner directory, download the runner installation package, extract the package, install
host dependencies, and symlink the necessary packages.

`configure.sh` - This script is not executed inside the base-image. It is necessary to execute it at startup
of the vm. This script registers the vm with GitHub and starts the runner service. The script pulls the
metadata properties injected by the VirtualBoxServer using the VBoxControl utility. These properties are
then used to configure and launch runner service.

`gha.service` - This is a service file that can be used to register the `configure.sh` script with systemd
so it executes at startup. This is used to dynamically register the runner with GitHub only once the vm
is online.

The generation and configuration of a VirtualBox VM Image is cover in the installation section below.

### Configuring the server

The VirtualBoxServer is executed as a simple binary. Users can download the latest binary from
the Releases pages on [GitHub](https://github.com/lindluni/runner-virtualbox/releases) or
they can use the Go toolchain to install the server locally:
`go get github.com/lindluni/runner-virtualbox/server@v1.0.0`

Users will need to make considerations about high-availability of the server. Users can create
a systemd service to manage the high-availability of the server. 

The VirtualBoxServer is based on the [Gin](https://github.com/gin-gonic/gin) framework. As a security
measure the VirtualBoxServer only implements the TLS protocol, it does not support unencrypted communication.
As such users need to generate either self-signed certificates using a utility like OpenSSL, or they need
to request certificates from a private or public certificate authority does as LetsEncrypt. As a convenience
this repository also contains a utility for generating the necessary TLS certificates. You can retrieve the
utility from the Releases pages on [GitHub](https://github.com/lindluni/runner-virtualbox/releases) or
they can use the Go toolchain to install the server locally:
`go get github.com/lindluni/runner-virtualbox/cert-generator@v1.0.0`

The generation and configuration of certificates are covered in the [installation](INSTALL.md) instructions.
