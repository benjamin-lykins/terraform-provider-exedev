terraform {
  required_providers {
    exedev = {
      source  = "benjamin-lykins/exedev"
      version = "~> 0.1"
    }
  }
}

provider "exedev" {}

data "exedev_vm" "existing" {
  name = "griffin-stone"
}

data "exedev_vm" "test_vm" {
  name = "tf-test-vm"
}

output "existing_hostname" {
  value = data.exedev_vm.existing.hostname
}

output "existing_status" {
  value = data.exedev_vm.existing.status
}

output "existing_image" {
  value = data.exedev_vm.existing.image
}

output "existing_region" {
  value = data.exedev_vm.existing.region
}

output "existing_disk" {
  value = data.exedev_vm.existing.disk
}

output "test_vm_hostname" {
  value = data.exedev_vm.test_vm.hostname
}

output "test_vm_status" {
  value = data.exedev_vm.test_vm.status
}
