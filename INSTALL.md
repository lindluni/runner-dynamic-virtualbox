
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

### Conclusion

That's it, you've configured VirtualBox to dynamically create runners on demand. You can extend the
examples in this repository to work for Mac, Windows, ARM or other Linux distros to suit your needs.