# terraform-provider-exedev

> **Warning:** This provider was developed with the assistance of AI and has not been fully tested or reviewed. This was intended as a side thing to see how quickly a minimally viable provider could be spun up. 

A Terraform provider for managing [exe.dev](https://exe.dev) resources via the [HTTPS API](https://exe.dev/docs/https-api).

## Requirements

- Terraform >= 1.0
- Go >= 1.22 (to build from source)

## Authentication

The provider authenticates using a Bearer token signed with your SSH private key.

### Generate a token

The provider requires a token with permissions to run `ls`, `new`, `rm`, `rename`, `resize`, `ssh-key list`, `ssh-key add`, `ssh-key remove`, and `ssh-key rename`. The default `exe0` token only includes a limited set — you must explicitly specify `cmds`:

```bash
# Add a dedicated API key (optional but recommended for revocability)
ssh-keygen -t ed25519 -C terraform -f ~/.ssh/exe_dev_terraform
cat ~/.ssh/exe_dev_terraform.pub | ssh exe.dev ssh-key add

# Generate the token with required permissions
export PERMISSIONS='{"cmds":["ls","new","rm","rename","resize","ssh-key list","ssh-key add","ssh-key remove","ssh-key rename"]}'
export PAYLOAD=$(printf '%s' "$PERMISSIONS" | base64 | tr -d '\n=' | tr '+/' '-_')
export SIG=$(printf '%s' "$PERMISSIONS" | ssh-keygen -Y sign -f ~/.ssh/exe_dev_terraform -n v0@exe.dev)
export SIGBLOB=$(echo "$SIG" | sed '1d;$d' | tr -d '\n=' | tr '+/' '-_')
export EXEDEV_TOKEN="exe0.$PAYLOAD.$SIGBLOB"
```

Alternatively, use `ssh-key generate-api-key` (requires an interactive SSH session):

```bash
ssh exe.dev ssh-key generate-api-key --label=terraform --exp=365d
```

## Provider Configuration

```hcl
terraform {
  required_providers {
    exedev = {
      source  = "benjamin-lykins/exedev"
      version = "~> 0.1"
    }
  }
}

provider "exedev" {
  # token can also be set via the EXEDEV_TOKEN environment variable
  token = var.exedev_token
}
```

## Resources

### `exedev_vm`

Manages an exe.dev virtual machine.

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

**Arguments:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | No | VM name. Auto-generated if omitted. |
| `image` | string | No | Container image (e.g. `ubuntu:22.04`). Forces replacement if changed. |
| `disk` | string | No | Disk size (e.g. `20GB`). Can be increased; decreasing forces replacement. |
| `command` | string | No | Container start command. Forces replacement if changed. |
| `env` | map(string) | No | Environment variables. Forces replacement if changed. |
| `integrations` | list(string) | No | Integration names to attach. Forces replacement if changed. |
| `setup_script` | string | No | Script to run on first boot. Forces replacement if changed. |

**Computed attributes:**
| Name | Description |
|------|-------------|
| `id` | VM name (same as `name`). |
| `hostname` | Public hostname (e.g. `my-vm.exe.xyz`). |
| `status` | Current VM status. |
| `region` | Region where the VM is running. |

### `exedev_ssh_key`

Manages an SSH key associated with your exe.dev account.

```hcl
resource "exedev_ssh_key" "ci" {
  public_key = "ssh-ed25519 AAAA... ci-bot"
  name       = "ci-bot"
}
```

**Arguments:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `public_key` | string | Yes | Full SSH public key string. Forces replacement if changed. |
| `name` | string | No | Key name. Defaults to the public key comment. Can be renamed in-place. |

**Computed attributes:**
| Name | Description |
|------|-------------|
| `id` | Key name. |
| `fingerprint` | SHA256 fingerprint of the key. |

## Data Sources

### `exedev_vm`

Reads information about an existing VM.

```hcl
data "exedev_vm" "existing" {
  name = "my-existing-vm"
}

output "existing_hostname" {
  value = data.exedev_vm.existing.hostname
}
```

## Import

VMs and SSH keys can be imported by name:

```bash
terraform import exedev_vm.web my-web-server
terraform import exedev_ssh_key.ci ci-bot
```

## Building from source

```bash
go build -o terraform-provider-exedev .
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/benjamin-lykins/exedev/0.1.0/darwin_arm64/
cp terraform-provider-exedev ~/.terraform.d/plugins/registry.terraform.io/benjamin-lykins/exedev/0.1.0/darwin_arm64/
```
