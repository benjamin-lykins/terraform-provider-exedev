terraform {
  required_providers {
    exedev = {
      source  = "benjamin-lykins/exedev"
      version = "~> 0.1"
    }
  }
}

provider "exedev" {
  # Set via EXEDEV_TOKEN environment variable or directly here (sensitive)
  # token = "exe0...."
}

variable "manage_ssh_key" {
  description = "Set true only if your token allows ssh-key add/remove/rename commands"
  type        = bool
  default     = false
}

variable "ssh_public_key" {
  description = "Real SSH public key to upload when manage_ssh_key is true"
  type        = string
  default     = ""
}

# Manage an SSH key for API/CI access
resource "exedev_ssh_key" "ci" {
  count = var.manage_ssh_key && var.ssh_public_key != "" ? 1 : 0

  public_key = var.ssh_public_key
  name       = "ci-bot"
}

# Create a VM
resource "exedev_vm" "app" {
  name  = "my-app"
  image = "ubuntu:22.04"
  disk  = "20GB"

  env = {
    APP_ENV = "production"
  }
}

# Look up an existing VM
# data "exedev_vm" "existing" {
#   name = "some-existing-vm"
# }

# output "app_hostname" {
#   description = "Hostname of the managed VM"
#   value       = exedev_vm.app.hostname
# }

# output "existing_hostname" {
#   description = "Hostname of the looked-up VM"
#   value       = data.exedev_vm.existing.hostname
# }
