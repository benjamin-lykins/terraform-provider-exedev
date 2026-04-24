# exedev_ssh_key Resource

Manages an SSH key associated with your exe.dev account.

## Example Usage

```hcl
resource "exedev_ssh_key" "ci" {
  public_key = "ssh-ed25519 AAAA... ci-bot"
  name       = "ci-bot"
}
```

## Argument Reference

* `public_key` - (Required) Full SSH public key string. Forces replacement if changed.
* `name` - (Optional) Key name. Defaults to the public key comment. Can be renamed in-place.

## Attribute Reference

* `id` - Key name.
* `fingerprint` - SHA256 fingerprint of the key.

## Import

SSH keys can be imported by name:

```shell
terraform import exedev_ssh_key.ci ci-bot
```
