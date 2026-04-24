# exedev_vm Resource

Manages a virtual machine on [exe.dev](https://exe.dev).

## Example Usage

```hcl
resource "exedev_vm" "web" {
  name  = "my-web-server"
  image = "ubuntu:22.04"
  disk  = "20GB"

  env = {
    APP_ENV = "production"
    PORT    = "8080"
  }

  integrations = ["my-github-integration"]
}

output "hostname" {
  value = exedev_vm.web.hostname
}
```

## Argument Reference

* `name` - (Optional) VM name. Auto-generated if omitted.
* `image` - (Optional) Container image (e.g. `ubuntu:22.04`). Forces replacement if changed.
* `disk` - (Optional) Disk size (e.g. `20GB`). Can be increased; decreasing forces replacement.
* `command` - (Optional) Container start command. Forces replacement if changed.
* `env` - (Optional) Map of environment variables. Forces replacement if changed.
* `integrations` - (Optional) List of integration names to attach. Forces replacement if changed.
* `setup_script` - (Optional) Script to run on first boot. Forces replacement if changed.

## Attribute Reference

* `id` - VM name (same as `name`).
* `hostname` - Public hostname (e.g. `my-vm.exe.xyz`).
* `status` - Current VM status.
* `region` - Region where the VM is running.

## Import

VMs can be imported by name:

```shell
terraform import exedev_vm.web my-web-server
```
