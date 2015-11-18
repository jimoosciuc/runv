package hypervisor

const (
	BaseDir         = "/var/run/hyper"
	HyperSockName   = "hyper.sock"
	TtySockName     = "tty.sock"
	ConsoleSockName = "console.sock"
	ShareDirTag     = "share_dir"
	DefaultKernel   = "/var/lib/hyper/kernel"
	DefaultInitrd   = "/var/lib/hyper/hyper-initrd.img"
	PciAddrFrom     = 0x05
	ExitChar        = 4
)

var InterfaceCount int = 1

const (
	EVENT_VM_START_FAILED = iota
	EVENT_VM_EXIT
	EVENT_VM_KILL
	EVENT_VM_TIMEOUT
	EVENT_POD_FINISH
	EVENT_INIT_CONNECTED
	EVENT_CONTAINER_ADD
	EVENT_CONTAINER_DELETE
	EVENT_VOLUME_ADD
	EVENT_VOLUME_DELETE
	EVENT_BLOCK_INSERTED
	EVENT_DEV_SKIP
	EVENT_BLOCK_EJECTED
	EVENT_INTERFACE_ADD
	EVENT_INTERFACE_DELETE
	EVENT_INTERFACE_INSERTED
	EVENT_INTERFACE_EJECTED
	EVENT_SERIAL_ADD
	EVENT_SERIAL_DELETE
	EVENT_TTY_OPEN
	EVENT_TTY_CLOSE
	COMMAND_GET_POD_IP
	COMMAND_RUN_POD
	COMMAND_REPLACE_POD
	COMMAND_STOP_POD
	COMMAND_SHUTDOWN
	COMMAND_RELEASE
	COMMAND_NEWCONTAINER
	COMMAND_EXEC
	COMMAND_KILL
	COMMAND_WRITEFILE
	COMMAND_READFILE
	COMMAND_ATTACH
	COMMAND_DETACH
	COMMAND_WINDOWSIZE
	COMMAND_ACK
	ERROR_INIT_FAIL
	ERROR_QMP_FAIL
	ERROR_INTERRUPTED
	ERROR_CMD_FAIL
)

const (
	INIT_RESERVED = iota
	INIT_STARTPOD
	INIT_GETPOD
	INIT_STOPPOD
	INIT_DESTROYPOD
	INIT_RESTARTCONTAINER
	INIT_EXECCMD
	INIT_FINISHCMD
	INIT_READY
	INIT_ACK
	INIT_ERROR
	INIT_WINSIZE
	INIT_PING
	INIT_FINISHPOD
	INIT_NEXT
	INIT_WRITEFILE
	INIT_READFILE
	INIT_NEWCONTAINER
	INIT_KILLCONTAINER
)

const (
	PREPARING_CONTAINER = iota
	PREPARING_VOLUME
	PREPARING_BLOCK
	PREPARING_BIND_DIR
)

func EventString(ev int) string {
	switch ev {
	case EVENT_VM_START_FAILED:
		return "EVENT_VM_START_FAILED"
	case EVENT_VM_EXIT:
		return "EVENT_VM_EXIT"
	case EVENT_VM_KILL:
		return "EVENT_VM_KILL"
	case EVENT_VM_TIMEOUT:
		return "EVENT_VM_TIMEOUT"
	case EVENT_POD_FINISH:
		return "EVENT_POD_FINISH"
	case EVENT_INIT_CONNECTED:
		return "EVENT_INIT_CONNECTED"
	case EVENT_CONTAINER_ADD:
		return "EVENT_CONTAINER_ADD"
	case EVENT_CONTAINER_DELETE:
		return "EVENT_CONTAINER_DELETE"
	case EVENT_VOLUME_ADD:
		return "EVENT_VOLUME_ADD"
	case EVENT_VOLUME_DELETE:
		return "EVENT_VOLUME_DELETE"
	case EVENT_DEV_SKIP:
		return "EVENT_DEV_SKIP"
	case EVENT_BLOCK_INSERTED:
		return "EVENT_BLOCK_INSERTED"
	case EVENT_BLOCK_EJECTED:
		return "EVENT_BLOCK_EJECTED"
	case EVENT_INTERFACE_ADD:
		return "EVENT_INTERFACE_ADD"
	case EVENT_INTERFACE_DELETE:
		return "EVENT_INTERFACE_DELETE"
	case EVENT_INTERFACE_INSERTED:
		return "EVENT_INTERFACE_INSERTED"
	case EVENT_INTERFACE_EJECTED:
		return "EVENT_INTERFACE_EJECTED"
	case EVENT_SERIAL_ADD:
		return "EVENT_SERIAL_ADD"
	case EVENT_SERIAL_DELETE:
		return "EVENT_SERIAL_DELETE"
	case EVENT_TTY_OPEN:
		return "EVENT_TTY_OPEN"
	case EVENT_TTY_CLOSE:
		return "EVENT_TTY_CLOSE"
	case COMMAND_GET_POD_IP:
		return "COMMAND_GET_POD_IP"
	case COMMAND_RUN_POD:
		return "COMMAND_RUN_POD"
	case COMMAND_REPLACE_POD:
		return "COMMAND_REPLACE_POD"
	case COMMAND_STOP_POD:
		return "COMMAND_STOP_POD"
	case COMMAND_SHUTDOWN:
		return "COMMAND_SHUTDOWN"
	case COMMAND_RELEASE:
		return "COMMAND_RELEASE"
	case COMMAND_NEWCONTAINER:
		return "COMMAND_NEWCONTAINER"
	case COMMAND_EXEC:
		return "COMMAND_EXEC"
	case COMMAND_WRITEFILE:
		return "COMMAND_WRITEFILE"
	case COMMAND_READFILE:
		return "COMMAND_READFILE"
	case COMMAND_ATTACH:
		return "COMMAND_ATTACH"
	case COMMAND_DETACH:
		return "COMMAND_DETACH"
	case COMMAND_WINDOWSIZE:
		return "COMMAND_WINDOWSIZE"
	case COMMAND_ACK:
		return "COMMAND_ACK"
	case ERROR_INIT_FAIL:
		return "ERROR_INIT_FAIL"
	case ERROR_QMP_FAIL:
		return "ERROR_QMP_FAIL"
	case ERROR_INTERRUPTED:
		return "ERROR_INTERRUPTED"
	case ERROR_CMD_FAIL:
		return "ERROR_CMD_FAIL"
	}
	return "UNKNOWN"
}
