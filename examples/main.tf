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

# Manage an SSH key for API/CI access
resource "exedev_ssh_key" "ci" {
  public_key = "ssh-ed25519 AAAA... ci-bot"
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
data "exedev_vm" "existing" {
  name = "some-existing-vm"
}

output "app_hostname" {
  description = "Hostname of the managed VM"
  value       = exedev_vm.app.hostname
}

output "existing_hostname" {
  description = "Hostname of the looked-up VM"
  value       = data.exedev_vm.existing.hostname
}
