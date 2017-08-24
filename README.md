[![Build Status](https://travis-ci.org/hyperhq/runv.svg?branch=master)](https://travis-ci.org/hyperhq/runv)

## ![runV](logo.png)

`runV` is a hypervisor-based runtime for [OCI](https://github.com/opencontainers/runtime-spec).

### OCI

`runV` is compatible with OCI. However, due to the difference between hypervisors and containers, the following sections of OCI don't apply to runV:
- Namespace
- Capability
- Device
- `linux` and `mount` fields in OCI specs are ignored

### Hypervisor

The current release of `runV` supports the following hypervisors:
- KVM (QEMU 2.1 or later)
- KVM (Kvmtool)
- Xen (4.5 or later)
- QEMU without KVM (NOT RECOMMENDED. QEMU 2.1 or later)

### Distro

The current release of `runV` supports the following distros:

- Ubuntu 64bit
	- 15.04 Vivid
	- 14.10 Utopic
	- 14.04 Trusty
- CentOS 64bit
	- 7.0
	- 6.x (upgrade to QEMU 2.1)
- Fedora 20-22 64bit
- Debian 64bit
	- 8.0 jessie
	- 7.x wheezy (upgrade to QEMU 2.1)

### Build

```bash
# install autoconf automake pkg-config make gcc golang qemu
# optional install device-mapper and device-mapper-devel for device-mapper storage
# optional install xen and xen-devel for xen driver
# optional install libvirt and libvirt-devel for libvirt driver
# note: the above package names might be different in various distros
# create a 'github.com/hyperhq' in your GOPATH/src
$ cd $GOPATH/src/github.com/hyperhq
$ git clone https://github.com/hyperhq/runv/
$ cd runv
$ ./autogen.sh
$ ./configure --without-xen
$ make
$ sudo make install
```

### Run

To run a OCI image, execute `runv` with the [OCI JSON format file](https://github.com/opencontainers/runc#oci-container-json-format) as argument, or have a `config.json` file in `CWD`.

Also, a kernel and initrd images are needed too. We recommend you to build them from [HyperStart](https://github.com/hyperhq/hyperstart/) repo. If not specified, runV will try to use the `/var/lib/hyper/kernel` and `/var/lib/hyper/hyper-initrd.img` files as the kernel and initrd images.

```bash
runv --kernel kernel --initrd initrd.img run mycontainer
$ ps aux
USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root         1  0.0  0.1   4352   232 ttyS0    S+   05:54   0:00 /init
root         2  0.0  0.5   4448   632 pts/0    Ss   05:54   0:00 sh
root         4  0.0  1.6  15572  2032 pts/0    R+   05:57   0:00 ps aux
```

### Run it with docker

`runv` is a runtime implementation of [OCI runtime](https://github.com/opencontainers/runtime-spec) and its command line is highly compatible with the 1.0.0-rc3(keeping updated with the newest released runc). But it is still under development and uncompleted.

`runV` provides [a detailed walk-though](docs/configure-runv-with-containerd-docker.md) to integrate with latest versions of docker and containerd.

Quick example (requires 17.06.1-ce that talks runc-1.0.0-rc3 command line):

Configure docker to use `runV` as the default runtime.
```bash
$cat /etc/docker/daemon.json
{
  "default-runtime": "runv",
  "runtimes": {
    "runv": {
      "path": "runv"
    }
  }
}
```

Start docker, pull and create busybox container.
```bash
$sudo systemctl start docker
$docker pull busybox
Using default tag: latest
latest: Pulling from library/busybox
Digest: sha256:2605a2c4875ce5eb27a9f7403263190cd1af31e48a2044d400320548356251c4
Status: Image is up to date for busybox:latest
$docker run --rm -it busybox
/ # ls
bin   dev   etc   home  lib   proc  root  sys   tmp   usr   var
/ # exit
```

### Example

Please follow the [instructions in runC](https://github.com/opencontainers/runc#creating-an-oci-bundle) to get the container rootfs and execute `runv spec` to generate a spec in the format of a `config.json` file.
