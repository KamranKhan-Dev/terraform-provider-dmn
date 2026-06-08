---
page_title: "dmn Provider"
description: |-
  Evaluate DMN decision tables inside Terraform.
---

# dmn Provider

The dmn provider evaluates DMN (Decision Model and Notation) decisions from standard `.dmn` files via the `provider::dmn::evaluate` function. Use it for multi-input, policy-driven lookups authored in a portable, standard format.

It exposes a single function and nothing else: no resources, no data sources, no state, and no provider configuration. You pass a DMN document and inputs, and it returns the decision's outputs — resolving any decisions and input data the target decision depends on (its decision requirements graph).

Requires Terraform >= 1.8 or OpenTofu >= 1.7 (provider-defined functions).

## Example Usage

```terraform
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
```

The result feeds straight into `data` sources and resources — the provider is a decision engine inside your plan, not just an output formatter. See the `evaluate` function documentation for the full signature, return shape, and argument details.

## When to use it

Terraform already handles trivial single-key lookups well with a native map (`lookup()` / `jsondecode(file(...))`), and that is simpler and preferred for "key → value". Reach for this provider when the decision logic stops fitting a flat map:

| Need | Map / JSON / CSV | DMN via this provider |
|------|------------------|------------------------|
| Multiple input dimensions | Deeply nested keys | Columns in one table |
| Ranges (`201–1000 rps`) | One entry per value | `(200..1000]` |
| "Any" / default | Manual fallbacks in HCL | `-` wildcard + FIRST/row order |
| Match several values | Duplicate rows | `"prod","dr"` list test |
| Exclusions | Inverted lookups | `not("prod")` |
| Computed outputs | Compute in HCL after lookup | FEEL output expressions (`rps / 50 + 2`) |
| Multiple results | Awkward to model | COLLECT → list → `for_each` |
| Chained logic | Intermediate locals | DRG (one decision feeds another) |
| Ownership | Buried in `.tf` | `.dmn` authored/reviewed by the owning team |

The decision table can be owned and edited by the team that owns the logic (in Camunda Modeler, Trisotech, …), reviewed as a diff, and reused outside Terraform.

## Supported DMN

- DMN **1.2 – 1.5** namespaces (decision-table structure and hit-policy semantics are stable across these versions).
- Decision logic: **decision tables** and **literal expressions**. Boxed expressions (context, invocation, BKM, list, relation, conditional, iterator) return a clear "unsupported" error.
- Full **decision requirements graph (DRG)** resolution for the target decision's dependencies, with cycle detection.
- Hit policies: UNIQUE, ANY, FIRST, PRIORITY, COLLECT (with sum/min/max/count), RULE ORDER, OUTPUT ORDER.
- Unary tests: equality, comparison operators (`<`, `<=`, `>`, `>=`), ranges (`[a..b]`, `(a..b)`, half-open), lists (`x, y, z`), `not(...)`, and the `-` wildcard.

~> **Note:** PRIORITY and OUTPUT ORDER currently behave as RULE ORDER (they require output `allowedValues` ordering, planned for a later release).

## Configuration

This provider takes no configuration. The bare `required_providers` entry shown above is all that is needed — there is no `provider "dmn" {}` block to populate.
