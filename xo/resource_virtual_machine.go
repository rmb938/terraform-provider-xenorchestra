package xo

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mitchellh/hashstructure"

	"github.com/rmb938/terraform-provider-xenorchestra/xo_client"
)

func resourceVirtualMachine() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVirtualMachineCreate,
		ReadContext:   resourceVirtualMachineRead,
		UpdateContext: resourceVirtualMachineUpdate,
		DeleteContext: resourceVirtualMachineDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"template_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cpus": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			// TODO: static and dynamic memory mins and max's
			"memory": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"installation": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"method": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice([]string{"network", "cd"}, false),
						},
						"disk_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"boot_disk": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"storage_repository_id": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"attached_disk": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 14,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"disk_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"device": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"position": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"network_interface": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attached": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"device": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"network_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"mac_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"desired_status": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"Running", "Halted"}, false),
			},
			"allow_stopping_for_update": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
		CustomizeDiff: customdiff.All(
			customdiff.ForceNewIfChange("boot_disk.0.size", func(ctx context.Context, old, new, meta interface{}) bool {
				return new.(int) < old.(int)
			}),
			func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
				// when creating an instance, name is not set
				oldName, _ := diff.GetChange("name")

				if oldName == nil || oldName == "" {
					_, newDesiredStatus := diff.GetChange("desired_status")

					if newDesiredStatus == nil || newDesiredStatus == "" {
						return nil
					} else if newDesiredStatus != "Running" {
						return fmt.Errorf("When creating an instance, desired_status can only accept Running value")
					}
					return nil
				}

				return nil
			},
		),
	}
}

func resourceVirtualMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	templateID := d.Get("template_id").(string)
	cpus := d.Get("cpus").(int)
	memory := d.Get("memory").(int)

	template, err := c.GetTemplateByID(ctx, templateID)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Error finding template with ID %s", templateID),
				Detail:   err.Error(),
			},
		}
	}

	var existingDisk *xo_client.VirtualMachineDisk
	var vmd *xo_client.VirtualMachineDisk
	var installation *xo_client.VirtualMachineInstallation

	bootDiskList := d.Get("boot_disk").([]interface{})
	bootDiskMap := bootDiskList[0].(map[string]interface{})
	bootDiskSRID := bootDiskMap["storage_repository_id"].(string)
	bootDiskSize := bootDiskMap["size"].(int)

	bootSR, err := c.GetStorageRepositoryByID(ctx, bootDiskSRID)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error finding storage repository for boot disk",
				Detail:   err.Error(),
			},
		}
	}

	if bootSR.Pool != template.Pool {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Boot Disk storage repository is not in the same pool as the template",
			},
		}
	}

	if bootSR.Type == "iso" {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Boot Disk storage repository cannot be of type ISO",
			},
		}
	}

	if len(template.VBDs) == 0 {
		installationList := d.Get("installation").([]interface{})
		if len(installationList) == 0 {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Template requires installation",
					Detail:   "The selected template requires installation fields to be set",
				},
			}
		}

		installationMap := installationList[0].(map[string]interface{})
		method := installationMap["method"].(string)
		VDIID := installationMap["disk_id"].(string)

		if method == "cd" && len(VDIID) == 0 {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "disk_id must be set when method is cd",
				},
			}
		}

		if method == "network" && len(VDIID) > 0 {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "disk_id cannot be set when method is network",
				},
			}
		}

		installation = &xo_client.VirtualMachineInstallation{
			Method: method,
		}

		if len(VDIID) > 0 {
			vdi, err := c.GetVDIByID(ctx, VDIID)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  fmt.Sprintf("Error finding VDI with ID %s for installation", VDIID),
						Detail:   err.Error(),
					},
				}
			}

			if vdi.Pool != template.Pool {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "Installation VDI is not in the same pool as the template",
					},
				}
			}

			sr, err := c.GetStorageRepositoryByID(ctx, vdi.StorageRepositoryID)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  fmt.Sprintf("Error finding storage repository from VDI with ID %s for installation", VDIID),
						Detail:   err.Error(),
					},
				}
			}

			if sr.Type != "iso" {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "VDI for installation is not in a storage repository with ISO type",
					},
				}
			}

			installation.Repository = vdi.ID
		}

		vmd = &xo_client.VirtualMachineDisk{
			Name:                "boot",
			StorageRepositoryID: bootSR.ID,
			Size:                bootDiskSize * 1024 * 1024 * 1024,
			Type:                "user",
		}
	} else {
		installationList := d.Get("installation").([]interface{})
		if len(installationList) != 0 {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Template does not support installation",
					Detail:   "The selected template does not support installation being set",
				},
			}
		}

		VBDs, err := template.GetVBDs(c, ctx, false)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error getting VBDs in template %s", templateID),
					Detail:   err.Error(),
				},
			}
		}

		if len(VBDs) != 1 {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Template (%s) doesn't have exactly 1 disk attached", templateID),
					Detail:   "The provider only supports creating VMs from templates with 1 disk attached",
				},
			}
		}

		vbd := VBDs[0]
		VDI, err := vbd.GetVDI(c, ctx)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error getting VDI (%s) from VBD %s", vbd.VDI, vbd.ID),
					Detail:   err.Error(),
				},
			}
		}

		vdiSizeGB := VDI.Size / 1024 / 1024 / 1024

		if bootDiskSize < vdiSizeGB {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Boot Disk Size needs to be equal or greater then the template disk size",
					Detail:   fmt.Sprintf("Template disk size is %d", vdiSizeGB),
				},
			}
		}

		existingDisk = &xo_client.VirtualMachineDisk{
			Name:                "boot",
			StorageRepositoryID: bootSR.ID,
			Size:                bootDiskSize * 1024 * 1024 * 1024,
		}
	}

	var networks []xo_client.Network

	networkInterfaceList := d.Get("network_interface").([]interface{})
	for _, networkInterface := range networkInterfaceList {
		networkInterfaceMap := networkInterface.(map[string]interface{})
		networkID := networkInterfaceMap["network_id"].(string)

		network, err := c.GetNetworkByID(ctx, networkID)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error getting Network %s", networkID),
					Detail:   err.Error(),
				},
			}
		}

		if network.Pool != template.Pool {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Network (%s) is not in the same pool as the template", network.ID),
				},
			}
		}

		networks = append(networks, *network)
	}

	virtualMachine, err := c.CreateVirtualMachine(
		ctx,
		name,
		description,
		template,
		cpus,
		memory*1024*1024*1024,
		installation,
		vmd,
		existingDisk,
		networks,
	)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error creating VM",
				Detail:   err.Error(),
			},
		}
	}

	d.SetId(virtualMachine.ID)

	attachDisksList := d.Get("attached_disk").([]interface{})
	for _, attachDisk := range attachDisksList {
		attachDiskMap := attachDisk.(map[string]interface{})
		vdiID := attachDiskMap["disk_id"].(string)

		vdi, err := c.GetVDIByID(ctx, vdiID)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error finding VDI with ID %s for attach disks", vdiID),
					Detail:   err.Error(),
				},
			}
		}

		err = virtualMachine.AttachDisk(c, ctx, vdi)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error attaching disk %s to VM", vdiID),
					Detail:   err.Error(),
				},
			}
		}
	}

	err = virtualMachine.Start(c, ctx)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error starting virtual machine",
				Detail:   err.Error(),
			},
		}
	}

	return resourceVirtualMachineRead(ctx, d, m)
}

func resourceVirtualMachineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	vm, err := c.GetVirtualMachineByID(ctx, d.Id())
	if err != nil {
		if err == xo_client.NotFoundError {
			d.SetId("")
			return nil
		}
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Error finding VM with ID %s", d.Id()),
				Detail:   err.Error(),
			},
		}
	}

	d.Set("name", vm.Name)
	d.Set("description", vm.Description)
	d.Set("cpus", vm.CPU.Max)
	d.Set("memory", vm.Memory.Static[1]/1024/1024/1024)

	var bootDiskList []map[string]interface{}

	bootVDI, err := vm.GetBootDisk(c, ctx)
	if err != nil {
		if err != xo_client.NotFoundError {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error finding boot disk vdi"),
					Detail:   err.Error(),
				},
			}
		}
	}

	if bootVDI != nil {
		bootDiskList = append(bootDiskList, map[string]interface{}{
			"storage_repository_id": bootVDI.StorageRepositoryID,
			"size":                  bootVDI.Size / 1024 / 1024 / 1024,
		})
	}

	d.Set("boot_disk", bootDiskList)

	var attachedDiskList []map[string]interface{}

	attachedVBDs, attchedVDIs, err := vm.GetAttachedDisks(c, ctx)
	if err != nil {
		if err != xo_client.NotFoundError {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error finding attached disks"),
					Detail:   err.Error(),
				},
			}
		}
	}

	for i, attachedDisk := range attchedVDIs {
		vbd := attachedVBDs[i]
		// disks are listed backwards so reverse the append
		attachedDiskList = append([]map[string]interface{}{
			{
				"disk_id":  attachedDisk.ID,
				"device":   vbd.Device,
				"position": vbd.Position,
			},
		}, attachedDiskList...)
	}

	d.Set("attached_disk", attachedDiskList)

	var networkInterfaceList []map[string]interface{}

	vifs, err := vm.GetVIFs(c, ctx)
	if err != nil {
		if err != xo_client.NotFoundError {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error finding vifs"),
					Detail:   err.Error(),
				},
			}
		}
	}

	for _, vif := range vifs {
		// vifs are listed backwards so reverse the append
		networkInterfaceList = append([]map[string]interface{}{
			{
				"attached":    vif.Attached,
				"device":      vif.Device,
				"network_id":  vif.NetworkID,
				"mac_address": vif.MAC,
			},
		}, networkInterfaceList...)
	}

	d.Set("network_interface", networkInterfaceList)

	if d.Get("desired_status") != "" {
		d.Set("desired_status", vm.PowerState)
	}

	return nil
}

func resourceVirtualMachineUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	var name *string
	var description *string

	allowStoppingForUpdate := d.Get("allow_stopping_for_update").(bool)
	stoppedForUpdate := false

	vm, err := c.GetVirtualMachineByID(ctx, d.Id())
	if err != nil {
		if err == xo_client.NotFoundError {
			d.SetId("")
			return nil
		}

		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting virtual machine",
				Detail:   err.Error(),
			},
		}
	}

	currentStatus := vm.PowerState

	if d.HasChange("name") {
		name = func(i string) *string { return &i }(d.Get("name").(string))
	}

	if d.HasChange("description") {
		name = func(i string) *string { return &i }(d.Get("description").(string))
	}

	err = vm.Update(c, ctx, name, description)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error updating virtual machine",
				Detail:   err.Error(),
			},
		}
	}

	bootDiskChanged := d.HasChange("boot_disk.0.size")
	attachDiskChanged := d.HasChange("attached_disk")
	networkChanged := d.HasChange("network_interface")

	// trying to change disks without pv drivers while running
	// this requires a power off
	if vm.PVDriversDetected == false && currentStatus == "Running" && (bootDiskChanged || attachDiskChanged || networkChanged) {

		// can't change attached disks when running
		if allowStoppingForUpdate == false {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Cannot change attached disks when VM is running without PV drivers unless allow_stopping_for_update is set to true",
				},
			}
		}

		err := vm.Stop(c, ctx, !vm.PVDriversDetected)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Error stopping VM for boot disk resize",
					Detail:   err.Error(),
				},
			}
		}
		stoppedForUpdate = true
	}

	if bootDiskChanged {
		size := d.Get("boot_disk.0.size").(int) * 1024 * 1024 * 1024
		bootVDI, err := vm.GetBootDisk(c, ctx)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Error getting boot disk",
					Detail:   err.Error(),
				},
			}
		}

		err = bootVDI.Update(c, ctx, nil, nil, &size)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Error expanding boot disk",
					Detail:   err.Error(),
				},
			}
		}
	}

	if attachDiskChanged {
		o, n := d.GetChange("attached_disk")
		vbds, err := vm.GetAttachedVBDs(c, ctx)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Error getting attached vbds for vm",
					Detail:   err.Error(),
				},
			}
		}

		// Keep track of disks currently in the instance. It's possible that there are fewer
		// disks currently attached than there were at the time we ran terraform plan.
		currDisks := map[string]xo_client.VBD{}
		for _, disk := range vbds {
			currDisks[disk.Position] = disk
		}

		// Keep track of disks currently in state.
		// Since changing any field within the disk needs to detach+reattach it,
		// keep track of the hash of the full disk.
		oDisks := map[uint64]xo_client.VBD{}
		for _, disk := range o.([]interface{}) {
			diskMap := disk.(map[string]interface{})
			position := diskMap["position"].(string)
			hash, err := hashstructure.Hash(diskMap, nil)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "Error hashing disk",
						Detail:   err.Error(),
					},
				}
			}

			if vbd, ok := currDisks[position]; ok {
				oDisks[hash] = vbd
			}
		}

		// Keep track of new config's disks.
		// Since changing any field within the disk needs to detach+reattach it,
		// keep track of the hash of the full disk.
		// If a disk with a certain hash is only in the new config, it should be attached.
		nDisks := map[uint64]struct{}{}
		var attach []map[string]interface{}
		for _, disk := range n.([]interface{}) {
			diskMap := disk.(map[string]interface{})
			hash, err := hashstructure.Hash(diskMap, nil)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "Error hashing disk",
						Detail:   err.Error(),
					},
				}
			}
			nDisks[hash] = struct{}{}

			if _, ok := oDisks[hash]; !ok {
				attach = append(attach, diskMap)
			}
		}

		// If a source is only in the old config, it should be detached.
		// Detach the old disks.
		for hash, vbd := range oDisks {
			if _, ok := nDisks[hash]; !ok {
				// if running we need to detach first
				if stoppedForUpdate == false && currentStatus == "Running" {
					err := vbd.Disconnect(c, ctx)
					if err != nil {
						return diag.Diagnostics{
							{
								Severity: diag.Error,
								Summary:  fmt.Sprintf("Error disconnecting vbd %s", vbd.ID),
								Detail:   err.Error(),
							},
						}
					}
				}

				err := vbd.Delete(c, ctx)
				if err != nil {
					return diag.Diagnostics{
						{
							Severity: diag.Error,
							Summary:  fmt.Sprintf("Error deleting vbd %s", vbd.ID),
							Detail:   err.Error(),
						},
					}
				}
			}
		}

		// Attach the new disks
		for _, diskMap := range attach {
			vdiID := diskMap["disk_id"].(string)

			vdi, err := c.GetVDIByID(ctx, vdiID)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  fmt.Sprintf("Error finding VDI with ID %s for attach disks", vdiID),
						Detail:   err.Error(),
					},
				}
			}

			err = vm.AttachDisk(c, ctx, vdi)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  fmt.Sprintf("Error attaching disk %s to VM", vdiID),
						Detail:   err.Error(),
					},
				}
			}
		}
	}

	if networkChanged {
		o, n := d.GetChange("network_interface")
		vifs, err := vm.GetVIFs(c, ctx)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Error getting attached vifs for vm",
					Detail:   err.Error(),
				},
			}
		}

		currVIFs := map[string]xo_client.VIF{}
		for _, vif := range vifs {
			currVIFs[vif.Device] = vif
		}

		oVIFs := map[uint64]xo_client.VIF{}
		for _, vif := range o.([]interface{}) {
			vifMap := vif.(map[string]interface{})
			position := vifMap["device"].(string)
			hash, err := hashstructure.Hash(vifMap, nil)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "Error hashing vif",
						Detail:   err.Error(),
					},
				}
			}

			if vif, ok := currVIFs[position]; ok {
				oVIFs[hash] = vif
			}
		}

		nVIFs := map[uint64]struct{}{}
		var attach []map[string]interface{}
		for _, vif := range n.([]interface{}) {
			vifMap := vif.(map[string]interface{})
			hash, err := hashstructure.Hash(vifMap, nil)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "Error hashing disk",
						Detail:   err.Error(),
					},
				}
			}
			nVIFs[hash] = struct{}{}

			if _, ok := oVIFs[hash]; !ok {
				attach = append(attach, vifMap)
			}
		}

		for hash, vif := range oVIFs {
			if _, ok := nVIFs[hash]; !ok {
				// if running we need to detach first
				if stoppedForUpdate == false && currentStatus == "Running" && vif.Attached {
					err := vif.Disconnect(c, ctx)
					if err != nil {
						return diag.Diagnostics{
							{
								Severity: diag.Error,
								Summary:  fmt.Sprintf("Error disconnecting vif %s", vif.ID),
								Detail:   err.Error(),
							},
						}
					}
				}

				err := vif.Delete(c, ctx)
				if err != nil {
					return diag.Diagnostics{
						{
							Severity: diag.Error,
							Summary:  fmt.Sprintf("Error deleting vif %s", vif.ID),
							Detail:   err.Error(),
						},
					}
				}
			}
		}

		for _, vifMap := range attach {
			networkID := vifMap["network_id"].(string)

			network, err := c.GetNetworkByID(ctx, networkID)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  fmt.Sprintf("Error finding network with ID %s", network),
						Detail:   err.Error(),
					},
				}
			}

			err = vm.AttachNetwork(c, ctx, network)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  fmt.Sprintf("Error creating VIF with network %s", network),
						Detail:   err.Error(),
					},
				}
			}
		}

	}

	desiredStatus := d.Get("desired_status")
	if stoppedForUpdate && desiredStatus == "" {
		err := vm.Start(c, ctx)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Error starting virtual machine after update",
					Detail:   err.Error(),
				},
			}
		}
	}

	if desiredStatus != "" && (stoppedForUpdate == true || vm.PowerState != desiredStatus) {
		if stoppedForUpdate == false && desiredStatus == "Halted" {
			err := vm.Stop(c, ctx, !vm.PVDriversDetected)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "Error stopping virtual machine",
						Detail:   err.Error(),
					},
				}
			}
		} else if desiredStatus == "Running" {
			err := vm.Start(c, ctx)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "Error starting virtual machine",
						Detail:   err.Error(),
					},
				}
			}
		}
	}

	return resourceVirtualMachineRead(ctx, d, m)
}

func resourceVirtualMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*xo_client.Client)

	vm, err := c.GetVirtualMachineByID(ctx, d.Id())
	if err != nil {
		if err == xo_client.NotFoundError {
			d.SetId("")
			return nil
		}

		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting virtual machine",
				Detail:   err.Error(),
			},
		}
	}

	if vm.PowerState != "Halted" {
		err := vm.Stop(c, ctx, !vm.PVDriversDetected)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "Error stopping virtual machine",
					Detail:   err.Error(),
				},
			}
		}
	}

	vbds, err := vm.GetAttachedVBDs(c, ctx)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error getting virtual machine vbds",
				Detail:   err.Error(),
			},
		}
	}

	for _, vbd := range vbds {
		err := vbd.Delete(c, ctx)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Error deleteing VBD %s from VM", vbd.ID),
					Detail:   err.Error(),
				},
			}
		}
	}

	err = vm.Delete(c, ctx)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Error deleting virtual machine",
				Detail:   err.Error(),
			},
		}
	}

	return nil
}
