terraform {
  required_providers {
    dmn = {
      source = "KamranKhan-Dev/dmn"
    }
  }
}

output "subnet" {
  # Single-result table: index [0] for the object, then the output column.
  value = provider::dmn::evaluate(file("${path.module}/networks.dmn"), "Network Lookup", {
    business_unit = "payments"
    environment   = "prod"
  })[0].subnet_id
}
