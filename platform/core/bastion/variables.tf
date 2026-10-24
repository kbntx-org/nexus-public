variable "hcloud_token" {
  description = "Hetzner Cloud API token"
  type        = string
  sensitive   = true
}

variable "cloudflare_api_token" {
  description = "Hetzner Cloud API token"
  type        = string
  sensitive   = true
}

variable "email_ssh_access" {
  description = "Email user allowed to access"
  type        = string
}
