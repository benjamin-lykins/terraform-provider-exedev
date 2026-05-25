# exedev_domain Resource

Registers a custom domain for an exe.dev VM.

Create the DNS record before applying this resource. For a subdomain, point a
DNS-only CNAME at the VM hostname (`vmname.exe.xyz`). exe.dev verifies DNS during
`domain add`; unregistered custom domains receive `421 Misdirected Request` even
when DNS points at the VM.

## Example Usage

```hcl
resource "exedev_vm" "web" {
  name  = "my-web-server"
  image = "ubuntu:22.04"
}

resource "exedev_domain" "web" {
  vm_name = exedev_vm.web.name
  domain  = "app.example.com"
}
```

## Argument Reference

* `vm_name` - (Required) exe.dev VM name that should serve the custom domain.
  Changing this forces a new resource.
* `domain` - (Required) Custom domain to register with exe.dev. Changing this
  forces a new resource.

## Attribute Reference

* `id` - Stable identifier in `vm_name/domain` form.

## Import

Domains can be imported with the VM name and domain:

```shell
terraform import exedev_domain.web my-web-server/app.example.com
```
