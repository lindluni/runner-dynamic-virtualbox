# VirtualBox Dynamic Provisioner

The VirtualBox Dynamic Provisioner (VBDP) is a client-server GitHub Action that allows users to
provision virtual machines using VirtualBox at workflow runtime. Users can control the
provisioning and destruction of vm's on any infrastructure where you can install VirtualBox.

This has major advantages over the GitHub hosted runners in that it allows you to bring any
custom hardware you have access to. If you require GPU-enabled vm's, or processors running
custom secure-enclave processors, such as Intel-SGX, users can make use of the VBDP to
enable their workflow to run on their custom hardware.

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

The generation and configuration of certificates are covered in the installation instructions below.

## Installation

This section provides an end-to-end example of generating a VirtualBox VM Image, configuring a VirtualBoxServer
to manage requests from a VirtualBoxClient, and instructions on generating a GitHub Action Workflow.

**NOTICE:** To get started, we will demonstrate running VirtualBox and the VirtualBoxServer on your local development machine.
As such you will need to configure you local router or gateway to forward requests from the client to your
local machine where the VirtualBoxServer is running. Please consult with your router or gateway manual for
configuring port-forwarding. If running the VirtualBoxServer on a bare-metal host in the cloud, you need only
apply the appropriate networking rules from your cloud provider to allow GitHub Actions to communicate with your
server. You can find a list of GitHub subnets to add to your ingress policies by querying the GitHub meta API, 
instructions for doing this can be found [here](https://docs.github.com/en/github/authenticating-to-github/about-githubs-ip-addresses).

### Prerequisites

The following prerequisites are required for running this example:

- [Oracle VirtualBox](https://www.virtualbox.org/)
- [Vagrant](https://www.vagrantup.com/)
- [VirtualBoxServer](https://github.com/lindluni/runner-virtualbox/releases)
- [Cert-Generator](https://github.com/lindluni/runner-virtualbox/releases)

### Creating a VM image

In this tutorial we will use Vagrant to manage the lifecycle of the VM that will serve as the base-image
of our runners.

Navigate to the directory where you wish to save your artifacts:

Create a working directory and initialize the Vagrantfile, we will be using the defacto standard Bento
Ubuntu Vagrant image for this tutorial:

```
mkdir -p vagrant/ubuntu/20.04
pushd vagrant/ubuntu/20.04
vagrant init bento/ubuntu-20.04
vagrant up
vagrant ssh
```

At this point, Vagrant will have created a vm using VirtualBox for you. The `vagrant ssh` command will
have put you inside the VM. We will now configure the vm by prepping the necessary file to dynamically
create a runner.

First we configure the service:

```
wget https://raw.githubusercontent.com/lindluni/runner-virtualbox/main/scripts/gha.service -P /etc/systemd/system/
wget https://raw.githubusercontent.com/lindluni/runner-virtualbox/main/scripts/configure.sh -P /home/vagrant
systemctl enable gha.service
```

This previous step starts by first installing the systemd service file `gha.service`. If you inspect it, you will 
see on startup it simply executes the `configure.sh` script we also downloaded as the `vagrant` user. Finally 
we execute `systemctl enable gha.service` which registers the service, and on the next restart, will 
execute `configure.sh`.

Next we setup all the necessary files and dependencies that will be used by `configure.sh` when it is executed
at startup:

```
wget https://raw.githubusercontent.com/lindluni/runner-virtualbox/main/scripts/setup.sh
./setup.sh
```

This step installs all the necessary APT dependencies and stages the GitHub Action Runner configuration and
launch scripts by executing `setup.sh`.

Now that we've configured the vm base-image, we can shutdown the host. When VirtualBox clones a machine
it is necessary that it be in a shutdown state. Execute the following commands to exit and shutdown the vm:

```
exit
vagrant halt
```

At this point you can open the VirtualBox application and right-click on your vm and rename it to something
more user-friendly, such as `gha`.

That's it for configuring the base-image. Next we will create the TLS certificates for the VirtualBoxServer

### Using cert-generator to generate TLS certificates

Cert-Generator is a simple command-line utility, it takes in two parameters and generates two files in
the current directory `cert.pem` and `key.pem` which are used by the VirtualBoxServer to secure communication.
The VirtualBoxClient performs certificate validation to ensure it is communicating with correct server. As such
we must provide the hostname or IP of the server where the VirtualBoxServer is hosted. To do this you can simply
visit [WhatIsMyIP](http://whatismyip.host/) and they will provide your external IP address. As mentioned
earlier at this point you need to configure port-forwarding on your router or gateway to forward from this
public IP address to your local machine.

Now that you have your IP you can run the Cert-Generator with the following command:
```
cert-generator -host <your IP> -org <any name for your organization> 
```

You will now have the `cert.pem` and `key.pem` in your current directory.

### Generate GitHub API token

Navigate to [GitHub Developer Setting](https://github.com/settings/tokens) and generate a new API token.
Take care to limit the scope of the token. It needs to be able to generate a Runner registration token, nothing
else. Once you've generated the token save it somewhere safe, such as a password vault.

### Running the VirtualBoxServer

Now that we've generated the TLS certs and retrieved a GitHub API token, its time to start this server.
From the same directory where we generated our certs you can execute the following command:

```
virtualboxserver --host 0.0.0.0 --port 8080 --cert-file cert.pem --key-file key.pem --token <GitHub API token>
```

That's it, your VirtualBoxServer is now running. It's time to create our first workflow using our new virtualbox
runners.

### Setup GitHub Secrets

First we will configure our GitHub secrets to secure the necessary information we need to run our workflows.
Navigate to the repository secrets page of the repository that will be executing the workflow:
`https://github.com/<owner>/<repo>/settings/secrets/actions`

You will need to create three secrets:

- `VIRTUALBOX_CERT` - Contains the contents of the `cert.pem` file we generated earlier
- `VIRTUALBOX_HOST` - The FQDN or IP where your VirtualBoxServer is hosted
- `VIRTUALBOX_PORT` - The port your VirtualBoxServer is listening on

### Create your workflow

You can now create a new GitHub workflow in the `.github/workflows` directory of repository. You can
read the `action.yml` file for an overview of the input parameter to `runner-virtualbox` action.

A simple workflow might look something like this, where you first create a vm, and after executing
your job, delete the vm regardless of whether your job fails or succeeds.

```
name: PullRequest
on:
  pull_request:
    branches: [ main ]
jobs:
  provision:
    runs-on: ubuntu-20.04
    steps:
      - name: Create VM
        uses: lindluni/runner-virtualbox@v0.0.1
        with:
          action: create
          prefix: vbx
          image:  gha
          host:   ${{ secrets.VIRTUALBOX_HOST }}
          port:   ${{ secrets.VIRTUALBOX_PORT }}
          cert:   ${{ secrets.VIRTUALBOX_CERT }}
          token:  ${{ secrets.GITHUB_TOKEN }}

  run:
    runs-on: vbx-${{ github.run_id }}
    needs: [provision]
    steps:
      - name: Test Runner      
        run: whoami
  
  deprovision:
    runs-on: ubuntu-latest
    needs: [run]
    if: always()
    steps:
      - name: Destroy VM
        uses: lindluni/runner-virtualbox@v0.0.1
        with:
          action: delete
          prefix: vbx
          host:   ${{ secrets.VIRTUALBOX_HOST }}
          port:   ${{ secrets.VIRTUALBOX_PORT }}
          cert:   ${{ secrets.VIRTUALBOX_CERT }}
          token:  ${{ secrets.GITHUB_TOKEN }}
```

Once you've committed and pushed the workflow to your main branch (or opened a PR with the changes),
you can checkout the `Actions` tab of your repo to see new PR's build on your VirtualBox vm's. If you
open a new PR and then open VirtualBox on your local machine you will see the vm's being created and
destroyed.

### In conclusion

That's it, you've configured VirtualBox to dynamically create runners on demand. You can extend the
examples in this repository to work for Mac, Windows, ARM or other Linux distros to suit your needs.