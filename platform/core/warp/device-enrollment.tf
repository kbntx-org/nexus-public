locals {
  human_enrollment_email      = "redacted@example.com"
  github_identity_provider_id = one([for provider in data.cloudflare_zero_trust_access_identity_providers.all.result : provider.id if provider.type == "github"])
}

data "cloudflare_zero_trust_access_identity_providers" "all" {
  account_id = local.cloudflare_account_id
}

resource "cloudflare_zero_trust_access_policy" "human_enrollment" {
  account_id       = local.cloudflare_account_id
  name             = "Require platform"
  decision         = "allow"
  session_duration = "24h"

  include = [{
    github_organization = {
      name                 = "kbntx-org"
      team                 = "platform"
      identity_provider_id = local.github_identity_provider_id
    }
  }]

  require = [{
    email = {
      email = local.human_enrollment_email
    }
  }]
}

resource "cloudflare_zero_trust_access_application" "warp" {
  account_id = local.cloudflare_account_id
  name       = "Warp Login App"
  type       = "warp"

  allowed_idps              = [local.github_identity_provider_id]
  auto_redirect_to_identity = true

  policies = [
    {
      id         = cloudflare_zero_trust_access_policy.human_enrollment.id
      precedence = 1
    },
    {
      id         = cloudflare_zero_trust_access_policy.ci_enrollment.id
      precedence = 2
    },
  ]
}
