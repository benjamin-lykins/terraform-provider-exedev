# exedev Provider

The `exedev` provider manages resources on [exe.dev](https://exe.dev) — a platform for running containerized virtual machines — via its [HTTPS API](https://exe.dev/docs/https-api).

## Example Usage

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

## Authentication

The provider authenticates using a Bearer token signed with your SSH private key.

### Generating a Token

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

## Argument Reference

* `token` - (Optional) Bearer token for the exe.dev HTTPS API. Can also be set via the `EXEDEV_TOKEN` environment variable.
