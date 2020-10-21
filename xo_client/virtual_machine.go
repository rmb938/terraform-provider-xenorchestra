package xo_client

import (
	"context"
)

type VirtualMachineVIF struct {
	NetworkID string `json:"network"`
	// TODO: mac
}

type VirtualMachineDisk struct {
	Name                string `json:"name_label"`
	Description         string `json:"name_description"`
	StorageRepositoryID string `json:"SR"`
	Size                int    `json:"size"`
	Type                string `json:"type"`
}

type VirtualMachineInstallation struct {
	Method     string `json:"method"`
	Repository string `json:"repository,omitempty"`
}

type VirtualMachineCPU struct {
	Max    int `json:"max"`
	Number int `json:"number"`
}

type VirtualMachineMemory struct {
	Dynamic []int `json:"dynamic"`
	Static  []int `json:"static"`
	Size    int   `json:"size"`
}

type VirtualMachine struct {
	ID                string               `json:"id"`
	Name              string               `json:"name_label"`
	Description       string               `json:"name_description"`
	CPU               VirtualMachineCPU    `json:"CPUs"`
	Memory            VirtualMachineMemory `json:"memory"`
	PowerState        string               `json:"power_state"`
	VIFs              []string             `json:"VIFs"`
	PVDriversDetected bool                 `json:"pvDriversDetected"`
	VBDs              []string             `json:"$VBDs"`
	Pool              string               `json:"$pool"`
}

func (c *Client) CreateVirtualMachine(ctx context.Context, name string, description string, template *Template, cpus, memory int, installation *VirtualMachineInstallation, vmd, existingDisk *VirtualMachineDisk, networks []Network) (*VirtualMachine, error) {

	vifs := make([]VirtualMachineVIF, 0)
	for _, network := range networks {
		vifs = append(vifs, VirtualMachineVIF{
			NetworkID: network.ID,
		})
	}

	params := map[string]interface{}{
		"bootAfterCreate":  false,
		"name_label":       name,
		"name_description": description,
		"template":         template.ID,
		"CPUs":             cpus,
		"memory":           memory,
		"VIFs":             vifs,
	}

	if installation != nil {
		params["installation"] = installation
	}

	if vmd != nil {
		params["VDIs"] = []VirtualMachineDisk{*vmd}
	}

	if existingDisk != nil {
		params["existingDisks"] = map[string]VirtualMachineDisk{
			"0": *existingDisk,
		}
	}

	var virtualMachineID string
	err := c.rpcConn.Call(ctx, "vm.create", params, &virtualMachineID)
	if err != nil {
		return nil, err
	}

	return c.GetVirtualMachineByID(ctx, virtualMachineID)
}

func (c *Client) GetVirtualMachineByID(ctx context.Context, id string) (*VirtualMachine, error) {
	query := ObjectQuery{
		"id": id,
	}

	objs, err := c.GetObjectsOfType(ctx, "VM", query)
	if err != nil {
		return nil, err
	}

	interf, err := objs.ConvertToSingle(VirtualMachine{})
	if err != nil {
		return nil, err
	}

	virtualMachine := interf.(VirtualMachine)
	return &virtualMachine, nil
}

func (vm *VirtualMachine) GetBootDisk(client *Client, ctx context.Context) (*VDI, error) {

	for _, vbdID := range vm.VBDs {
		vbd, err := client.GetVBDByID(ctx, vbdID)
		if err != nil {
			return nil, err
		}

		// ignore cd drives
		if vbd.CDDrive == true {
			continue
		}

		// boot drives are always position 0 so ignore ones not here
		if vbd.Position != "0" {
			continue
		}

		vdi, err := vbd.GetVDI(client, ctx)
		if err != nil {
			return nil, err
		}

		return vdi, nil
	}

	return nil, NotFoundError
}

func (vm *VirtualMachine) GetAttachedVBDs(client *Client, ctx context.Context) ([]VBD, error) {
	var vbds []VBD

	for _, vbdID := range vm.VBDs {
		vbd, err := client.GetVBDByID(ctx, vbdID)
		if err != nil {
			return nil, err
		}

		// ignore cd drives
		if vbd.CDDrive == true {
			continue
		}

		// ignore position 0 drives since those are boot drives
		if vbd.Position == "0" {
			continue
		}

		vbds = append(vbds, *vbd)
	}

	return vbds, nil
}

func (vm *VirtualMachine) GetAttachedDisks(client *Client, ctx context.Context) ([]VBD, []VDI, error) {
	var vbds []VBD
	var vdis []VDI

	for _, vbdID := range vm.VBDs {
		vbd, err := client.GetVBDByID(ctx, vbdID)
		if err != nil {
			return nil, nil, err
		}

		// ignore cd drives
		if vbd.CDDrive == true {
			continue
		}

		// ignore position 0 drives since those are boot drives
		if vbd.Position == "0" {
			continue
		}

		vdi, err := vbd.GetVDI(client, ctx)
		if err != nil {
			return nil, nil, err
		}

		vbds = append(vbds, *vbd)
		vdis = append(vdis, *vdi)
	}

	return vbds, vdis, nil
}

func (vm *VirtualMachine) GetVIFs(client *Client, ctx context.Context) ([]VIF, error) {
	var vifs []VIF

	for _, vifID := range vm.VIFs {
		vif, err := client.GetVIFByID(ctx, vifID)
		if err != nil {
			return nil, err
		}

		vifs = append(vifs, *vif)
	}

	return vifs, nil
}

func (vm *VirtualMachine) AttachDisk(client *Client, ctx context.Context, vdi *VDI) error {
	params := map[string]interface{}{
		"vdi": vdi.ID,
		"vm":  vm.ID,
	}

	return client.rpcConn.Call(ctx, "vm.attachDisk", params, nil)
}

func (vm *VirtualMachine) Update(client *Client, ctx context.Context, name, description *string) error {
	params := map[string]interface{}{
		"id": vm.ID,
	}

	if name != nil {
		params["name_label"] = name
	}

	if description != nil {
		params["name_description"] = description
	}

	return client.rpcConn.Call(ctx, "vm.set", params, nil)
}

func (vm *VirtualMachine) Delete(client *Client, ctx context.Context) error {
	params := map[string]interface{}{
		"id": vm.ID,
	}

	return client.rpcConn.Call(ctx, "vm.delete", params, nil)
}

func (vm *VirtualMachine) Stop(client *Client, ctx context.Context, force bool) error {
	params := map[string]interface{}{
		"id":    vm.ID,
		"force": force,
	}

	return client.rpcConn.Call(ctx, "vm.stop", params, nil)
}

func (vm *VirtualMachine) Start(client *Client, ctx context.Context) error {
	params := map[string]interface{}{
		"id": vm.ID,
	}

	return client.rpcConn.Call(ctx, "vm.start", params, nil)
}
