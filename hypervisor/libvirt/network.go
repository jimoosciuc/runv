package libvirt

import (
	"os"

	"github.com/hyperhq/runv/hypervisor/network"
	"github.com/hyperhq/runv/hypervisor/pod"
)

func (ld *LibvirtDriver) BuildinNetwork() bool {
	return false
}

func (ld *LibvirtDriver) InitNetwork(bIface, bIP string) error {
	return nil
}

func (lc *LibvirtContext) ConfigureNetwork(vmId, requestedIP string,
	maps []pod.UserContainerPort, config pod.UserInterface) (*network.Settings, error) {
	return network.Configure(vmId, requestedIP, true, maps, config)
}

func (lc *LibvirtContext) AllocateNetwork(vmId, requestedIP string,
	maps []pod.UserContainerPort) (*network.Settings, error) {
	return network.Allocate(vmId, requestedIP, true, maps)
}

func (lc *LibvirtContext) ReleaseNetwork(vmId, releasedIP string, maps []pod.UserContainerPort,
	file *os.File) error {
	return network.Release(vmId, releasedIP, maps, nil)
}
