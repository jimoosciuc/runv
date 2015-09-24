package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/hyperhq/runv/driverloader"
	"github.com/hyperhq/runv/hypervisor"
	"github.com/hyperhq/runv/hypervisor/pod"
	"github.com/hyperhq/runv/hypervisor/types"
	"github.com/hyperhq/runv/lib/term"
	"github.com/opencontainers/specs"
)

const shortLen = 12

func TruncateID(id string) string {
	trimTo := shortLen
	if len(id) < shortLen {
		trimTo = len(id)
	}
	return id[:trimTo]
}

// GenerateRandomID returns an unique id
func GenerateRandomID() string {
	for {
		id := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, id); err != nil {
			panic(err) // This shouldn't happen
		}
		value := hex.EncodeToString(id)
		// if we try to parse the truncated for as an int and we don't have
		// an error then the value is all numberic and causes issues when
		// used as a hostname. ref #3869
		if _, err := strconv.ParseInt(TruncateID(value), 10, 64); err == nil {
			continue
		}
		return value
	}
}

func resizeTty(vm *hypervisor.Vm, tag string, outFd uintptr, isTerminalOut bool) {
	height, width := getTtySize(outFd, isTerminalOut)
	if height == 0 && width == 0 {
		return
	}

	vm.Tty(tag, height, width)
}

func monitorTtySize(vm *hypervisor.Vm, tag string, outFd uintptr, isTerminalOut bool) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGWINCH)
	go func() {
		for range sigchan {
			resizeTty(vm, tag, outFd, isTerminalOut)
		}
	}()
}

func getTtySize(outFd uintptr, isTerminalOut bool) (int, int) {
	if !isTerminalOut {
		return 0, 0
	}
	ws, err := term.GetWinsize(outFd)
	if err != nil {
		fmt.Printf("Error getting size: %s", err.Error())
		if ws == nil {
			return 0, 0
		}
	}
	return int(ws.Height), int(ws.Width)
}

func startVContainer(context *cli.Context) {
	hypervisor.InterfaceCount = 0

	var containerInfoList []*hypervisor.ContainerInfo
	var roots []string
	var containerId string
	var err error

	ocffile := context.String("config-file")
	runtimefile := context.String("runtime-file")

	if _, err = os.Stat(ocffile); os.IsNotExist(err) {
		fmt.Printf("Please specify ocffile or put config.json under current working directory\n")
		return
	}

	vbox := context.GlobalString("vbox")
	if _, err = os.Stat(vbox); err == nil {
		vbox, err = filepath.Abs(vbox)
		if err != nil {
			fmt.Printf("Cannot get abs path for vbox: %s\n", err.Error())
			return
		}
	}

	kernel := context.GlobalString("kernel")
	if _, err = os.Stat(kernel); err == nil {
		kernel, err = filepath.Abs(kernel)
		if err != nil {
			fmt.Printf("Cannot get abs path for kernel: %s\n", err.Error())
			return
		}
	}

	initrd := context.GlobalString("initrd")
	if _, err = os.Stat(initrd); err == nil {
		initrd, err = filepath.Abs(initrd)
		if err != nil {
			fmt.Printf("Cannot get abs path for initrd: %s\n", err.Error())
			return
		}
	}

	driver := context.GlobalString("driver")
	if hypervisor.HDriver, err = driverloader.Probe(driver); err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	podId := fmt.Sprintf("pod-%s", pod.RandStr(10, "alpha"))
	vmId := fmt.Sprintf("vm-%s", pod.RandStr(10, "alpha"))

	ocfData, err := ioutil.ReadFile(ocffile)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	var runtimeData []byte = nil
	_, err = os.Stat(runtimefile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Fail to stat %s, %s\n", runtimefile, err.Error())
			return
		}
	} else {
		runtimeData, err = ioutil.ReadFile(runtimefile)
		if err != nil {
			fmt.Printf("Fail to readfile %s, %s\n", runtimefile, err.Error())
			return
		}
	}

	userPod, err := pod.OCFConvert2Pod(ocfData, runtimeData)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	mypod := hypervisor.NewPod(podId, userPod)

	var (
		cpu = 1
		mem = 128
	)

	if userPod.Resource.Vcpu > 0 {
		cpu = userPod.Resource.Vcpu
	}

	if userPod.Resource.Memory > 0 {
		mem = userPod.Resource.Memory
	}

	b := &hypervisor.BootConfig{
		Kernel: kernel,
		Initrd: initrd,
		Bios:   "",
		Cbfs:   "",
		Vbox:   vbox,
		CPU:    cpu,
		Memory: mem,
	}

	vm := hypervisor.NewVm(vmId, cpu, mem, false, types.VM_KEEP_NONE)
	err = vm.Launch(b)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	sharedDir := path.Join(hypervisor.BaseDir, vm.Id, hypervisor.ShareDirTag)

	for _, c := range userPod.Containers {
		var root string
		var err error

		containerId = GenerateRandomID()
		rootDir := path.Join(sharedDir, containerId)
		os.MkdirAll(rootDir, 0755)

		rootDir = path.Join(rootDir, "rootfs")

		if !filepath.IsAbs(c.Image) {
			root, err = filepath.Abs(c.Image)
			if err != nil {
				fmt.Printf("%s\n", err.Error())
				return
			}
		} else {
			root = c.Image
		}

		err = mount(root, rootDir)
		if err != nil {
			fmt.Printf("mount %s to %s failed: %s\n", root, rootDir, err.Error())
			return
		}
		roots = append(roots, rootDir)

		containerInfo := &hypervisor.ContainerInfo{
			Id:     containerId,
			Rootfs: "rootfs",
			Image:  containerId,
			Fstype: "dir",
		}

		containerInfoList = append(containerInfoList, containerInfo)
		mypod.AddContainer(containerId, podId, "", []string{}, types.S_POD_CREATED)
	}

	qemuResponse := vm.StartPod(mypod, userPod, containerInfoList, nil)
	if qemuResponse.Data == nil {
		fmt.Printf("StartPod fail: QEMU response data is nil\n")
		return
	}
	fmt.Printf("result: code %d %s\n", qemuResponse.Code, qemuResponse.Cause)

	inFd, _ := term.GetFdInfo(os.Stdin)
	outFd, isTerminalOut := term.GetFdInfo(os.Stdout)

	oldState, err := term.SetRawTerminal(inFd)
	if err != nil {
		return
	}

	height, width := getTtySize(outFd, isTerminalOut)
	winSize := &hypervisor.WindowSize{
		Row:    uint16(height),
		Column: uint16(width),
	}

	tag := pod.RandStr(8, "alphanum")

	monitorTtySize(vm, tag, outFd, isTerminalOut)

	vm.Attach(os.Stdin, os.Stdout, tag, containerId, winSize)

	qemuResponse = vm.StopPod(mypod, "yes")

	term.RestoreTerminal(inFd, oldState)

	for _, root := range roots {
		umount(root)
	}

	if qemuResponse.Data == nil {
		fmt.Printf("StopPod fail: QEMU response data is nil\n")
		return
	}
	fmt.Printf("result: code %d %s\n", qemuResponse.Code, qemuResponse.Cause)
}

var startCommand = cli.Command{
	Name:  "start",
	Usage: "create and run a container",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "config-file, c",
			Value: "config.json",
			Usage: "path to spec config file",
		},
		cli.StringFlag{
			Name:  "runtime-file, r",
			Value: "runtime.json",
			Usage: "path to runtime config file",
		},
	},
	Action: startVContainer,
}

var specCommand = cli.Command{
	Name:  "spec",
	Usage: "create a new specification file",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "config-file, c",
			Value: "config.json",
			Usage: "path to spec config file for writing",
		},
		cli.StringFlag{
			Name:  "runtime-file, r",
			Value: "runtime.json",
			Usage: "path to runtime config file for writing",
		},
	},
	Action: func(context *cli.Context) {
		spec := specs.Spec{
			Version: specs.Version,
			Platform: specs.Platform{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
			},
			Root: specs.Root{
				Path:     "rootfs",
				Readonly: true,
			},
			Process: specs.Process{
				Terminal: true,
				User:     specs.User{},
				Args: []string{
					"sh",
				},
				Env: []string{
					"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					"TERM=xterm",
				},
			},
			Hostname: "shell",
		}

		rspec := specs.LinuxRuntimeSpec{
			Linux: specs.LinuxRuntime{
				Resources: &specs.Resources{},
			},
		}

		checkNoFile := func(name string) error {
			_, err := os.Stat(name)
			if err == nil {
				return fmt.Errorf("File %s exists. Remove it first", name)
			}
			if !os.IsNotExist(err) {
				return err
			}
			return nil
		}

		cName := context.String("config-file")
		rName := context.String("runtime-file")
		if err := checkNoFile(cName); err != nil {
			fmt.Printf("%s", err.Error())
			return
		}
		if err := checkNoFile(rName); err != nil {
			fmt.Printf("%s", err.Error())
			return
		}
		data, err := json.MarshalIndent(&spec, "", "\t")
		if err != nil {
			fmt.Printf("%s", err.Error())
			return
		}
		if err := ioutil.WriteFile(cName, data, 0666); err != nil {
			fmt.Printf("%s", err.Error())
			return
		}
		rdata, err := json.MarshalIndent(&rspec, "", "\t")
		if err != nil {
			fmt.Printf("%s", err.Error())
			return
		}
		if err := ioutil.WriteFile(rName, rdata, 0666); err != nil {
			fmt.Printf("%s", err.Error())
			return
		}
	},
}
