variable "project" {
  type = string
  description = "Vault Azure auth method test with Azure Function"
  default     = "vlt-auth-func"
}

variable "env" {
  type    = string
  default = "test"
}

variable "region" {
  type        = string
  description = "The region to create Azure resources in"
  default     = "westus2"
}
