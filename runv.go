package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
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

const (
	_ = iota
	RUNV_ACK
	RUNV_EXECCMD
	RUNV_WINSIZE
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

func removeState(podId, root string, sock net.Listener) {
	os.RemoveAll(path.Join(root, podId))
	sock.Close()
}

func saveState(vmId, podId, root string) (net.Listener, *os.File, error) {
	podPath := path.Join(root, podId)

	_, err := os.Stat(podPath)
	if err == nil {
		return nil, nil, fmt.Errorf("Container %s exists\n", podId)
	}

	err = os.MkdirAll(podPath, 0644)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, nil, err
	}

	defer func() {
		if err != nil {
			os.RemoveAll(podPath)
		}
	}()

	pwd, err := filepath.Abs(".")
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, nil, err
	}

	state := specs.State{
		Version: specs.Version,
		ID:      vmId,
		Pid:     -1,
		Root:    pwd,
	}

	stateData, err := json.MarshalIndent(&state, "", "\t")
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, nil, err
	}

	stateFile := path.Join(podPath, "state.json")
	err = ioutil.WriteFile(stateFile, stateData, 0644)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, nil, err
	}

	stateFd, err := os.Open(stateFile)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, nil, err
	}

	sock, err := net.Listen("unix", path.Join(podPath, "runv.sock"))
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, nil, err
	}

	return sock, stateFd, nil
}

func execHook(hook specs.Hook, state *os.File) error {
	old_stdin, err := syscall.Dup(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("dup stdin failed, %s\n", err.Error())
		return err
	}

	defer syscall.Close(old_stdin)
	err = syscall.Dup2(int(state.Fd()), int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("dup stdin to state file failed, %s\n", err.Error())
		return err
	}
	defer syscall.Dup2(old_stdin, int(os.Stdin.Fd()))

	pAttr := &os.ProcAttr{
		Env:   hook.Env,
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr, state},
	}

	p, err := os.StartProcess(hook.Path, hook.Args, pAttr)
	if err != nil {
		fmt.Printf("start hook process failed, %s\n", err.Error())
		return err
	}

	status, err := p.Wait()
	if err != nil {
		fmt.Printf("wait hook process exit failed, %s\n", err.Error())
		return err
	}

	if status.Success() {
		return nil
	}

	return fmt.Errorf("execute hook %s failed", hook.Path)
}

func execPrestartHooks(rt *specs.RuntimeSpec, state *os.File) error {
	if rt == nil {
		return nil
	}

	for _, hook := range rt.Hooks.Prestart {
		err := execHook(hook, state)
		if err != nil {
			return err
		}
	}

	return nil
}

func execPoststartHooks(rt *specs.RuntimeSpec, state *os.File) error {
	if rt == nil {
		return nil
	}

	for _, hook := range rt.Hooks.Prestart {
		err := execHook(hook, state)
		if err != nil {
			fmt.Printf("exec Poststart hook %s failed %s", hook.Path, err.Error())
		}
	}

	return nil
}

func execPoststopHooks(rt *specs.RuntimeSpec, state *os.File) error {
	if rt == nil {
		return nil
	}

	for _, hook := range rt.Hooks.Prestart {
		err := execHook(hook, state)
		if err != nil {
			fmt.Printf("exec Poststop hook %s failed %s", hook.Path, err.Error())
		}
	}

	return nil
}

func parseUserPod(context *cli.Context) (*pod.UserPod, *specs.RuntimeSpec, error) {
	ocffile := context.String("config-file")
	runtimefile := context.String("runtime-file")

	if _, err := os.Stat(ocffile); os.IsNotExist(err) {
		fmt.Printf("Please specify ocffile or put config.json under current working directory\n")
		return nil, nil, err
	}

	ocfData, err := ioutil.ReadFile(ocffile)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, nil, err
	}

	var runtimeData []byte = nil
	_, err = os.Stat(runtimefile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Fail to stat %s, %s\n", runtimefile, err.Error())
			return nil, nil, err
		}
	} else {
		runtimeData, err = ioutil.ReadFile(runtimefile)
		if err != nil {
			fmt.Printf("Fail to readfile %s, %s\n", runtimefile, err.Error())
			return nil, nil, err
		}
	}

	userPod, rt, err := pod.OCFConvert2Pod(ocfData, runtimeData)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, nil, err
	}

	return userPod, rt, nil
}

func preparePod(mypod *hypervisor.PodStatus, userPod *pod.UserPod, vmId string) ([]*hypervisor.ContainerInfo, error) {
	var (
		containerInfoList []*hypervisor.ContainerInfo
		containerId       string
	)

	sharedDir := path.Join(hypervisor.BaseDir, vmId, hypervisor.ShareDirTag)

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
				return nil, err
			}
		} else {
			root = c.Image
		}

		err = mount(root, rootDir)
		if err != nil {
			fmt.Printf("mount %s to %s failed: %s\n", root, rootDir, err.Error())
			return nil, err
		}

		containerInfo := &hypervisor.ContainerInfo{
			Id:     containerId,
			Rootfs: "rootfs",
			Image:  containerId,
			Fstype: "dir",
		}

		containerInfoList = append(containerInfoList, containerInfo)
		mypod.AddContainer(containerId, mypod.Id, "", []string{}, types.S_POD_CREATED)
	}

	return containerInfoList, nil
}

func startVm(context *cli.Context, userPod *pod.UserPod, vmId string, sock net.Listener) (*hypervisor.Vm, error) {
	var (
		err error
		cpu = 1
		mem = 128
	)

	vbox := context.GlobalString("vbox")
	if _, err = os.Stat(vbox); err == nil {
		vbox, err = filepath.Abs(vbox)
		if err != nil {
			fmt.Printf("Cannot get abs path for vbox: %s\n", err.Error())
			return nil, err
		}
	}

	kernel := context.GlobalString("kernel")
	if _, err = os.Stat(kernel); err == nil {
		kernel, err = filepath.Abs(kernel)
		if err != nil {
			fmt.Printf("Cannot get abs path for kernel: %s\n", err.Error())
			return nil, err
		}
	}

	initrd := context.GlobalString("initrd")
	if _, err = os.Stat(initrd); err == nil {
		initrd, err = filepath.Abs(initrd)
		if err != nil {
			fmt.Printf("Cannot get abs path for initrd: %s\n", err.Error())
			return nil, err
		}
	}

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
	err = LaunchOCIVm(vm, b, sock)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, err
	}

	return vm, nil
}

func startVContainer(context *cli.Context) {
	var (
		err      error
		Response *types.VmResponse
	)

	hypervisor.InterfaceCount = 0

	driver := context.GlobalString("driver")
	if hypervisor.HDriver, err = driverloader.Probe(driver); err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	root := context.GlobalString("root")

	podId := context.GlobalString("id")
	vmId := fmt.Sprintf("vm-%s", pod.RandStr(10, "alpha"))

	fmt.Printf("runv container id: %s\n", podId)
	sock, stateFd, err := saveState(vmId, podId, root)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	defer removeState(podId, root, sock)

	userPod, rt, err := parseUserPod(context)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	mypod := hypervisor.NewPod(podId, userPod)
	infoList, err := preparePod(mypod, userPod, vmId)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	defer func() {
		tmpDir := path.Join(hypervisor.BaseDir, vmId)
		for _, c := range mypod.Containers {
			rootDir := path.Join(tmpDir, hypervisor.ShareDirTag, c.Id, "rootfs")
			umount(rootDir)
		}
		os.RemoveAll(tmpDir)
	}()

	vm, err := startVm(context, userPod, vmId, sock)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	defer func() {
		Response = vm.StopPod(mypod, "yes")

		if Response.Data == nil {
			fmt.Printf("StopPod fail: QEMU response data is nil\n")
			return
		}
		fmt.Printf("result: code %d %s\n", Response.Code, Response.Cause)
	}()

	err = execPrestartHooks(rt, stateFd)
	if err != nil {
		fmt.Printf("execute Prestart hooks failed, %s\n", err.Error())
		return
	}

	inFd, _ := term.GetFdInfo(os.Stdin)
	outFd, isTerminalOut := term.GetFdInfo(os.Stdout)

	oldState, err := term.SetRawTerminal(inFd)
	if err != nil {
		return
	}

	defer term.RestoreTerminal(inFd, oldState)

	tag := pod.RandStr(8, "alphanum")
	ttyCallback := make(chan *types.VmResponse, 1)

	// using pipes in vm.Attach to avoid the stdio to be closed
	fromStd, toVm := io.Pipe()
	fromVm, toStd := io.Pipe()
	go io.Copy(toVm, os.Stdin)
	go io.Copy(os.Stdout, fromVm)

	err = vm.Attach(fromStd, toStd, tag, mypod.Containers[0].Id, ttyCallback, nil)
	if err != nil {
		fmt.Printf("StartPod fail: fail to set up tty connection.\n")
		return
	}

	Response = vm.StartPod(mypod, userPod, infoList, nil)
	if Response.Data == nil {
		fmt.Printf("StartPod fail: QEMU response data is nil\n")
		return
	}
	fmt.Printf("result: code %d %s\n", Response.Code, Response.Cause)

	err = execPoststartHooks(rt, stateFd)
	if err != nil {
		fmt.Printf("execute Poststart hooks failed %s\n", err.Error())
	}

	newTty(vm, path.Join(root, podId), tag, outFd, isTerminalOut).monitorTtySize()
	<-ttyCallback

	err = execPoststopHooks(rt, stateFd)
	if err != nil {
		fmt.Printf("execute Poststop hooks failed %s\n", err.Error())
		return
	}
}

func runvGetTag(conn net.Conn) (tag string, err error) {
	msg, err := hypervisor.ReadVmMessage(conn.(*net.UnixConn))
	if err != nil {
		fmt.Printf("read runv server data failed: %v\n", err)
		return "", err
	}

	if msg.Code != RUNV_ACK {
		return "", fmt.Errorf("unexpected respond code")
	}

	return string(msg.Message), nil
}

func HandleRunvRequest(vm *hypervisor.Vm, conn net.Conn) {
	defer conn.Close()

	msg, err := hypervisor.ReadVmMessage(conn.(*net.UnixConn))
	if err != nil {
		fmt.Printf("read runv client data failed: %v\n", err)
		return
	}

	switch msg.Code {
	case RUNV_EXECCMD:
		{
			tag := pod.RandStr(8, "alphanum")
			m := &hypervisor.DecodedMessage{
				Code:    RUNV_ACK,
				Message: []byte(tag),
			}
			data := hypervisor.NewVmMessage(m)
			conn.Write(data)

			fmt.Printf("client exec cmd request %s\n", msg.Message[:])
			err = vm.Exec(conn, conn, string(msg.Message[:]), tag, vm.Pod.Containers[0].Id)

			if err != nil {
				fmt.Printf("read runv client data failed: %v\n", err)
			}
		}
	case RUNV_WINSIZE:
		{
			var winSize ttyWinSize
			json.Unmarshal(msg.Message, &winSize)
			//fmt.Printf("client exec winsize request %v\n", winSize)
			vm.Tty(winSize.Tag, winSize.Height, winSize.Width)
		}
	default:
		fmt.Printf("unknown cient request\n")
	}
}

func LaunchOCIVm(vm *hypervisor.Vm, b *hypervisor.BootConfig, sock net.Listener) error {
	err := vm.Launch(b)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := sock.Accept()
			if err != nil {
				fmt.Printf("accept on runv Socket err: %v\n", err)
				break
			}

			go HandleRunvRequest(vm, conn)
		}
	}()

	return nil
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
