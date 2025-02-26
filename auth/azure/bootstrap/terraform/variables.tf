# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "region" {
  type        = string
  description = "The region to create Azure resources in"
  default     = "westus2"
}

variable "ssh_public_key_path" {
  type        = string
  description = "Path to an SSH public key which should be used for authentication"
  default     = "~/.ssh/id_rsa.pub"
}