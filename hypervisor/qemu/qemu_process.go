package qemu

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/hyperhq/runv/hypervisor"
)

func watchDog(qc *QemuContext, hub chan hypervisor.VmEvent) {
	wdt := qc.wdt
	for {
		msg, ok := <-wdt
		if ok {
			switch msg {
			case "quit":
				glog.V(1).Info("quit watch dog.")
				return
			case "kill":
				success := false
				if qc.process != nil {
					glog.V(0).Infof("kill Qemu... %d", qc.process.Pid)
					if err := qc.process.Kill(); err == nil {
						success = true
					}
				} else {
					glog.Warning("no process to be killed")
				}
				hub <- &hypervisor.VmKilledEvent{Success: success}
				return
			}
		} else {
			glog.V(1).Info("chan closed, quit watch dog.")
			break
		}
	}
}

func (qc *QemuContext) watchPid(pid int, hub chan hypervisor.VmEvent) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	qc.process = proc
	go watchDog(qc, hub)

	return nil
}

// launchQemu run qemu and wait it's quit, includes
func launchQemu(qc *QemuContext, ctx *hypervisor.VmContext) {
	qemu := qc.driver.executable
	if qemu == "" {
		ctx.Hub <- &hypervisor.VmStartFailEvent{Message: "can not find qemu executable"}
		return
	}

	args := qc.arguments(ctx)

	if glog.V(1) {
		glog.Info("cmdline arguments: ", strings.Join(args, " "))
	}

	cmd := exec.Command("qemu-system-x86_64", args...)
	err := cmd.Run()
	if err != nil {
		//fail to daemonize
		glog.Errorf("%v", err)
		ctx.Hub <- &hypervisor.VmStartFailEvent{Message: "try to start qemu failed"}
		return
	}

	var file *os.File
	t := time.NewTimer(time.Second * 5)
	// keep opening file until it exists or timeout
	for {
		select {
		case <-t.C:
			glog.Error("open pid file timeout")
			ctx.Hub <- &hypervisor.VmStartFailEvent{Message: "pid file not exist, timeout"}
			return
		default:
		}

		if file, err = os.OpenFile(qc.qemuPidFile, os.O_RDONLY, 0640); err != nil {
			file.Close()
			if os.IsNotExist(err) {
				continue
			}
			glog.Errorf("open pid file failed: %v", err)
			ctx.Hub <- &hypervisor.VmStartFailEvent{Message: "open pid file failed"}
			return
		}
		break
	}

	var pid uint32
	t = time.NewTimer(time.Second * 5)
	for {
		select {
		case <-t.C:
			glog.Error("read pid file timeout")
			ctx.Hub <- &hypervisor.VmStartFailEvent{Message: "read pid file timeout"}
			return
		default:
		}

		file.Seek(0, os.SEEK_SET)
		if _, err := fmt.Fscan(file, &pid); err != nil {
			if err == io.EOF {
				continue
			}
			glog.Errorf("read pid file failed: %v", err)
			ctx.Hub <- &hypervisor.VmStartFailEvent{Message: "read pid file failed"}
			return
		}
		break
	}

	file.Close()

	glog.V(1).Infof("starting daemon with pid: %d", pid)

	err = ctx.DCtx.(*QemuContext).watchPid(int(pid), ctx.Hub)
	if err != nil {
		glog.Error("watch qemu process failed")
		ctx.Hub <- &hypervisor.VmStartFailEvent{Message: "watch qemu process failed"}
		return
	}
}

func associateQemu(ctx *hypervisor.VmContext) {
	go watchDog(ctx.DCtx.(*QemuContext), ctx.Hub)
}
