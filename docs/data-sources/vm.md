# exedev_vm Data Source

Reads information about an existing exe.dev virtual machine.

## Example Usage

```hcl
data "exedev_vm" "existing" {
  name = "my-existing-vm"
}

output "existing_hostname" {
  value = data.exedev_vm.existing.hostname
}
```

## Argument Reference

* `name` - (Required) The name of the VM to look up.

## Attribute Reference

* `id` - VM name.
* `hostname` - Public hostname (e.g. `my-vm.exe.xyz`).
* `status` - Current VM status.
* `region` - Region where the VM is running.
* `image` - Container image the VM was created with.
* `disk` - Disk size of the VM.
