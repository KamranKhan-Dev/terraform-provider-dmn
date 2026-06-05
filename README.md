# terraform-provider-dmn

A Terraform provider that evaluates DMN (Decision Model and Notation) decision tables: pass inputs, get decision outputs directly in your Terraform workflow.

It exposes a single function, `provider::dmn::evaluate`, which parses a standard `.dmn` file and evaluates a named decision — resolving any decisions and input data it depends on — and returns the result. No resources, no state, no provider configuration.

## When to use it

Terraform already handles trivial single-key lookups well with a map:

```hcl
locals { net = jsondecode(file("networks.json")) }
# subnet_id = local.net[var.business_unit].subnet_id
```

Reach for this provider when a map stops being a good fit:

- **Multiple inputs with conditions or ranges** (e.g. `region`, `tier`, and `requests_per_second between …`).
- **Hit policies** — UNIQUE, ANY, FIRST, PRIORITY, COLLECT (with sum/min/max/count), RULE ORDER, OUTPUT ORDER.
- **Decision logic owned outside Terraform**, authored and maintained by platform or business teams in standard DMN tools (Camunda Modeler, Trisotech, …) and version-controlled as `.dmn`.

### Why not just a map / JSON / CSV?

A flat map answers "key → value". Real platform decisions are rarely that shape. Consider selecting a landing-zone subnet from `business_unit × environment × workload`, where some cells are wildcards and some depend on a numeric range. As a nested map this becomes a combinatorial sprawl you must hand-maintain for every combination:

```hcl
# Every business unit × every environment × every workload, fully enumerated,
# with no way to express "any environment" or "201–500 endpoints" without
# duplicating rows. Add one workload type and you touch dozens of entries.
locals {
  subnets = {
    payments = {
      prod = { default = "subnet-a", data = "subnet-b", pe-heavy = "subnet-c" }
      dev  = { default = "subnet-d", data = "subnet-d", pe-heavy = "subnet-d" }
      # ...repeat for qa, staging, dr...
    }
    # ...repeat the whole block for every business unit...
  }
}
```

DMN expresses the *same* logic declaratively, with wildcards, ranges, lists, and ordering built in — and the table can be owned and edited by the network team in a DMN modeler, reviewed as a diff, and reused outside Terraform. The examples below show what that buys you.

## Requirements

- Terraform >= 1.8 or OpenTofu >= 1.7 (provider-defined functions).

```hcl
terraform {
  required_providers {
    dmn = {
      source = "KamranKhan-Dev/dmn"
    }
  }
}
```

## Quick reference

```text
evaluate(definitions string, decision_name string, inputs dynamic) dynamic
```

- `definitions` — the DMN XML, typically via `file("decision.dmn")`.
- `decision_name` — the `name` (or `id`) of the decision to evaluate.
- `inputs` — an object of input variables keyed by the decision's input names.

The function **always returns a list** of result entries, so callers index `[0]` for single-result tables. Each entry is an object of named outputs (or the bare value for a single unnamed output). Single-result hit policies (UNIQUE/ANY/FIRST/PRIORITY) return a 0- or 1-element list; multi-result policies (COLLECT/RULE ORDER/OUTPUT ORDER) return N entries; COLLECT aggregations return a single-element list with the scalar; no match returns an empty list.

---

## Example 1 — Azure: Private Endpoint subnet selection (the platform-team classic)

Application teams shouldn't need to know VNET/subnet topology. They tag their workload with a **business unit**, **environment**, and **workload class**, and the platform team's DMN table returns the right network placement. This is the canonical "decouple app teams from network internals" use case.

The decision table (`landing_zone.dmn`, authored in Camunda Modeler), hit policy **FIRST** so the first matching row wins and a wildcard `-` row acts as a default:

| # | business_unit | environment | workload (`workload`) | subnet_name | vnet_name | network_rg |
|---|---------------|-------------|-----------------------|-------------|-----------|------------|
| 1 | `"payments"`  | `"prod"`    | `"pe-heavy"`          | `"snet-pe-pay-prod"` | `"vnet-pay-prod"` | `"rg-net-pay-prod"` |
| 2 | `"payments"`  | `"prod"`    | `-`                   | `"snet-app-pay-prod"` | `"vnet-pay-prod"` | `"rg-net-pay-prod"` |
| 3 | `"payments"`  | `not("prod")` | `-`                 | `"snet-app-pay-nonprod"` | `"vnet-pay-nonprod"` | `"rg-net-pay-nonprod"` |
| 4 | `-`           | `-`         | `-`                   | `"snet-shared-default"` | `"vnet-shared"` | `"rg-net-shared"` |

FEEL features a JSON map can't express cleanly: the `-` **wildcard**, `not("prod")` **negation**, and **row ordering** (FIRST) that turns the last row into a catch-all default.

**Feeding a `data` block** — the result drives data-source lookups and a real resource:

```hcl
locals {
  placement = provider::dmn::evaluate(
    file("${path.module}/landing_zone.dmn"),
    "Subnet Selection",
    {
      business_unit = var.business_unit
      environment   = var.environment
      workload      = var.workload_class
    },
  )[0] # FIRST → single object
}

# Output of the decision initializes data sources, not just outputs:
data "azurerm_resource_group" "network" {
  name = local.placement.network_rg
}

data "azurerm_subnet" "endpoint" {
  name                 = local.placement.subnet_name
  virtual_network_name = local.placement.vnet_name
  resource_group_name  = data.azurerm_resource_group.network.name
}

resource "azurerm_private_endpoint" "sql" {
  name                = "pe-${var.app_name}-sql"
  location            = var.location
  resource_group_name = var.app_resource_group
  subnet_id           = data.azurerm_subnet.endpoint.id # resolved by the DMN table

  private_service_connection {
    name                           = "psc-${var.app_name}-sql"
    private_connection_resource_id = azurerm_mssql_server.this.id
    subresource_names              = ["sqlServer"]
    is_manual_connection           = false
  }
}
```

The app team supplies three tags; the platform team owns the table. Neither hard-codes a subnet ID.

---

## Example 2 — AWS: instance & capacity sizing from numeric ranges

Sizing decisions depend on *ranges*, which maps can't represent without enumerating every value. Here a service's tier is chosen from **expected requests/second** and **environment**, and the outputs include **FEEL expressions** computed from the inputs (not just constants).

`sizing.dmn`, hit policy **UNIQUE**:

| # | environment | rps (`requests_per_second`) | instance_type | min_capacity | max_capacity |
|---|-------------|-----------------------------|---------------|--------------|--------------|
| 1 | `"prod"`    | `[0..200]`                  | `"m6i.large"`   | `2` | `requests_per_second / 50 + 2` |
| 2 | `"prod"`    | `(200..1000]`               | `"m6i.xlarge"`  | `3` | `requests_per_second / 50 + 2` |
| 3 | `"prod"`    | `> 1000`                    | `"m6i.2xlarge"` | `6` | `requests_per_second / 40 + 4` |
| 4 | `not("prod")` | `-`                       | `"t3.medium"`   | `1` | `2` |

FEEL features here: **half-open ranges** (`(200..1000]` excludes 200, includes 1000), **open-ended comparisons** (`> 1000`), and **output expressions** that do arithmetic over the inputs (`requests_per_second / 50 + 2`) — the table returns *computed* capacity, not a lookup constant.

```hcl
locals {
  sizing = provider::dmn::evaluate(
    file("${path.module}/sizing.dmn"),
    "Service Sizing",
    {
      environment         = var.environment
      requests_per_second = var.expected_rps
    },
  )[0]
}

# Drive a data lookup for the AMI matching the chosen instance family:
data "aws_ami" "app" {
  most_recent = true
  owners      = ["self"]

  filter {
    name   = "name"
    values = ["app-base-${split(".", local.sizing.instance_type)[0]}-*"]
  }
}

resource "aws_autoscaling_group" "app" {
  name             = "asg-${var.app_name}"
  min_size         = local.sizing.min_capacity
  max_size         = local.sizing.max_capacity
  desired_capacity = local.sizing.min_capacity
  # launch template references data.aws_ami.app.id and local.sizing.instance_type
}
```

To express row 2 (`201–1000 rps → m6i.xlarge`) in a map you'd need an entry per integer. DMN states the range once.

---

## Example 3 — AWS: multi-AZ subnet fan-out with COLLECT

Some decisions return *several* answers. With the **COLLECT** hit policy the function returns every matching row, which pairs naturally with `for_each`. Here a workload's subnets are selected across availability zones using a **FEEL list** to match multiple environments with one row.

`az_subnets.dmn`, hit policy **COLLECT**:

| # | business_unit | environment | az | subnet_id |
|---|---------------|-------------|------|-----------|
| 1 | `"payments"`  | `"prod","dr"` | `"euw1-az1"` | `"subnet-0aaa"` |
| 2 | `"payments"`  | `"prod","dr"` | `"euw1-az2"` | `"subnet-0bbb"` |
| 3 | `"payments"`  | `"prod","dr"` | `"euw1-az3"` | `"subnet-0ccc"` |

`"prod","dr"` is a FEEL **list test** (matches either value) — one row covers two environments.

```hcl
locals {
  # COLLECT → a list of {az, subnet_id} objects
  az_subnets = provider::dmn::evaluate(
    file("${path.module}/az_subnets.dmn"),
    "AZ Subnets",
    {
      business_unit = var.business_unit
      environment   = var.environment
    },
  )
}

# One data lookup per returned subnet — the decision drives the fan-out:
data "aws_subnet" "selected" {
  for_each = { for s in local.az_subnets : s.az => s.subnet_id }
  id       = each.value
}

resource "aws_lb" "app" {
  name               = "alb-${var.app_name}"
  load_balancer_type = "application"
  subnets            = [for s in data.aws_subnet.selected : s.id]
}
```

A map returns one value per key; COLLECT returns a *set* of rows that you can iterate — multi-result lookups without contorting the data structure.

---

## Example 4 — DRG composition: one decision feeding another

DMN decisions can depend on other decisions (a Decision Requirements Graph). The provider resolves the whole graph: it evaluates the prerequisite decision first and feeds its output into the next. A flat map has no equivalent — you'd compute intermediate values yourself in HCL.

`account_routing.dmn` contains two linked decisions. **Tier** classifies the workload; **Target Account** uses the resulting `tier` to pick the AWS account and OU. The `tier` input of the second table is *produced* by the first decision, not supplied by the caller.

Decision **Tier** (UNIQUE):

| # | data_classification | tier |
|---|---------------------|------|
| 1 | `"restricted","confidential"` | `"regulated"` |
| 2 | `-` | `"standard"` |

Decision **Target Account** (UNIQUE), requires *Tier*:

| # | business_unit | tier (from *Tier*) | account_id | org_unit |
|---|---------------|--------------------|------------|----------|
| 1 | `"payments"`  | `"regulated"`      | `"1111-2222-3333"` | `"ou-payments-regulated"` |
| 2 | `"payments"`  | `"standard"`       | `"4444-5555-6666"` | `"ou-payments-standard"` |

```hcl
locals {
  routing = provider::dmn::evaluate(
    file("${path.module}/account_routing.dmn"),
    "Target Account", # the provider evaluates "Tier" first, automatically
    {
      business_unit       = var.business_unit
      data_classification = var.data_classification
    },
  )[0]
}

# The composed decision result initializes an Organizations lookup:
data "aws_organizations_organizational_unit" "target" {
  parent_id = var.org_root_id
  name      = local.routing.org_unit
}

provider "aws" {
  alias  = "target"
  assume_role {
    role_arn = "arn:aws:iam::${replace(local.routing.account_id, "-", "")}:role/Provisioner"
  }
}
```

The caller never computes `tier` — the DMN graph does. Change the classification rules once, in the *Tier* table, and every consumer follows.

---

## Putting it together: why it's richer than a map/JSON/CSV

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

And in every example the decision result feeds straight into `data` sources and resources — the provider is a decision engine inside your plan, not just an output formatter.

## Supported DMN

- DMN **1.2 – 1.5** namespaces (decision-table structure and hit-policy semantics are stable across these versions).
- Decision logic: **decision tables** and **literal expressions**. Boxed expressions (context, invocation, BKM, list, relation, conditional, iterator) return a clear "unsupported" error.
- Full **decision requirements graph (DRG)** resolution for the target decision's dependencies, with cycle detection.
- Unary tests: equality, comparison operators (`<`, `<=`, `>`, `>=`), ranges (`[a..b]`, `(a..b)`, half-open), lists (`x, y, z`), `not(...)`, and the `-` wildcard. Numeric ranges/comparisons are supported in v1.

FEEL expression evaluation is provided by the MIT-licensed [`pbinitiative/feel`](https://github.com/pbinitiative/feel), wrapped behind an internal interface; decision-table unary-test semantics are implemented in this provider.

> Note: PRIORITY and OUTPUT ORDER currently behave as RULE ORDER (they require output `allowedValues` ordering, planned for a later release).

## Development

```bash
make build    # go build ./...
make test     # unit tests
make testacc  # TF_ACC=1 acceptance tests (requires a Terraform >= 1.8 binary)
make fmt vet  # format and vet
```

## License

MIT. See [LICENSE](LICENSE).
