package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/hyperhq/runv/containerd/api/grpc/types"
	"github.com/hyperhq/runv/lib/linuxsignal"
	"github.com/urfave/cli"
	netcontext "golang.org/x/net/context"
)

var linuxSignalMap = map[string]syscall.Signal{
	"ABRT":   linuxsignal.SIGABRT,
	"ALRM":   linuxsignal.SIGALRM,
	"BUS":    linuxsignal.SIGBUS,
	"CHLD":   linuxsignal.SIGCHLD,
	"CLD":    linuxsignal.SIGCLD,
	"CONT":   linuxsignal.SIGCONT,
	"FPE":    linuxsignal.SIGFPE,
	"HUP":    linuxsignal.SIGHUP,
	"ILL":    linuxsignal.SIGILL,
	"INT":    linuxsignal.SIGINT,
	"IO":     linuxsignal.SIGIO,
	"IOT":    linuxsignal.SIGIOT,
	"KILL":   linuxsignal.SIGKILL,
	"PIPE":   linuxsignal.SIGPIPE,
	"POLL":   linuxsignal.SIGPOLL,
	"PROF":   linuxsignal.SIGPROF,
	"PWR":    linuxsignal.SIGPWR,
	"QUIT":   linuxsignal.SIGQUIT,
	"SEGV":   linuxsignal.SIGSEGV,
	"STKFLT": linuxsignal.SIGSTKFLT,
	"STOP":   linuxsignal.SIGSTOP,
	"SYS":    linuxsignal.SIGSYS,
	"TERM":   linuxsignal.SIGTERM,
	"TRAP":   linuxsignal.SIGTRAP,
	"TSTP":   linuxsignal.SIGTSTP,
	"TTIN":   linuxsignal.SIGTTIN,
	"TTOU":   linuxsignal.SIGTTOU,
	"UNUSED": linuxsignal.SIGUNUSED,
	"URG":    linuxsignal.SIGURG,
	"USR1":   linuxsignal.SIGUSR1,
	"USR2":   linuxsignal.SIGUSR2,
	"VTALRM": linuxsignal.SIGVTALRM,
	"WINCH":  linuxsignal.SIGWINCH,
	"XCPU":   linuxsignal.SIGXCPU,
	"XFSZ":   linuxsignal.SIGXFSZ,
}

type killContainerCmd struct {
	Name   string
	Root   string
	Signal syscall.Signal
}

var killCommand = cli.Command{
	Name:  "kill",
	Usage: "kill sends the specified signal (default: SIGTERM) to the container's init process",
	ArgsUsage: `<container-id> <signal>

Where "<container-id>" is the name for the instance of the container and
"<signal>" is the signal to be sent to the init process.

For example, if the container id is "ubuntu01" the following will send a "KILL"
signal to the init process of the "ubuntu01" container:

       # runv kill ubuntu01 KILL`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "send the signal to all processes in the container",
		},
	},
	Action: func(context *cli.Context) error {
		container := context.Args().First()
		if container == "" {
			return cli.NewExitError("container id cannot be empty", -1)
		}

		sigstr := context.Args().Get(1)
		if sigstr == "" {
			sigstr = "SIGTERM"
		}
		signal, err := parseSignal(sigstr)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("parse signal failed %v, signal string:%s", err, sigstr), -1)
		}

		c, err := getClient(filepath.Join(context.GlobalString("root"), container, "namespace/namespaced.sock"))
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("failed to get client: %v", err), -1)
		}

		plist := make([]string, 0)

		if context.Bool("all") {
			if plist, err = getProcessList(c, container); err != nil {
				return cli.NewExitError(fmt.Sprintf("can't get process list, %v", err), -1)
			}
		} else {
			plist = append(plist, "init")
		}

		for _, p := range plist {
			if _, err = c.Signal(netcontext.Background(), &types.SignalRequest{
				Id:     container,
				Pid:    p,
				Signal: uint32(signal),
			}); err != nil {
				return cli.NewExitError(fmt.Sprintf("kill signal failed, %v", err), -1)
			}
		}
		return nil
	},
}

func getProcessList(c types.APIClient, container string) ([]string, error) {
	s, err := c.State(netcontext.Background(), &types.StateRequest{Id: container})
	if err != nil {
		return nil, fmt.Errorf("get container state failed, %v", err)
	}

	for _, cc := range s.Containers {
		if cc.Id == container {
			plist := make([]string, 0)
			for _, p := range cc.Processes {
				plist = append(plist, p.Pid)
			}

			return plist, nil
		}
	}

	return nil, fmt.Errorf("container %s not found", container)
}

func parseSignal(rawSignal string) (syscall.Signal, error) {
	s, err := strconv.Atoi(rawSignal)
	if err == nil {
		return syscall.Signal(s), nil
	}
	signal, ok := linuxSignalMap[strings.TrimPrefix(strings.ToUpper(rawSignal), "SIG")]
	if !ok {
		return -1, fmt.Errorf("unknown signal %q", rawSignal)
	}
	return signal, nil
}
