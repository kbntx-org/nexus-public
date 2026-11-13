data "cloudflare_zero_trust_organization" "this" {
  account_id = local.cloudflare_account_id
}

locals {
  internal_ingress_address = "10.43.216.131"
  private_network_ranges   = ["10.0.0.0/16", "10.42.0.0/16", "10.43.0.0/16"]
  ci_identity              = "identity.email == \"non_identity@${data.cloudflare_zero_trust_organization.this.auth_domain}\""
}

resource "cloudflare_zero_trust_access_service_token" "ci" {
  account_id = local.cloudflare_account_id
  name       = "GitHub Actions"
  duration   = "8760h"
}

resource "cloudflare_zero_trust_access_policy" "ci_enrollment" {
  account_id = local.cloudflare_account_id
  name       = "GitHub Actions enrollment"
  decision   = "non_identity"

  include = [{
    service_token = {
      token_id = cloudflare_zero_trust_access_service_token.ci.id
    }
  }]
}

resource "cloudflare_zero_trust_gateway_policy" "ci_allow_registry" {
  account_id = local.cloudflare_account_id
  name       = "GitHub Actions: allow the registry"
  action     = "allow"
  enabled    = true
  precedence = 1000
  filters    = ["l4"]
  identity   = local.ci_identity
  traffic    = "net.dst.ip == ${local.internal_ingress_address} and net.dst.port == 443"
}

resource "cloudflare_zero_trust_gateway_policy" "ci_block_private_network" {
  account_id = local.cloudflare_account_id
  name       = "GitHub Actions: block the private network"
  action     = "block"
  enabled    = true
  precedence = 1001
  filters    = ["l4"]
  identity   = local.ci_identity
  traffic    = "net.dst.ip in {${join(" ", local.private_network_ranges)}}"

  depends_on = [cloudflare_zero_trust_gateway_policy.ci_allow_registry]
}

output "ci_client_id" {
  value = cloudflare_zero_trust_access_service_token.ci.client_id
}

output "ci_client_secret" {
  value     = cloudflare_zero_trust_access_service_token.ci.client_secret
  sensitive = true
}
