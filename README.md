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

- **Multiple inputs with conditions or ranges** (e.g. `region`, `tier`, and `amount between …`).
- **Hit policies** — UNIQUE, ANY, FIRST, PRIORITY, COLLECT (with sum/min/max/count), RULE ORDER, OUTPUT ORDER.
- **Decision logic owned outside Terraform**, authored and maintained by platform or business teams in standard DMN tools (Camunda Modeler, Trisotech, …) and version-controlled as `.dmn`.

## Requirements

- Terraform >= 1.8 or OpenTofu >= 1.7 (provider-defined functions).

## Usage

```hcl
terraform {
  required_providers {
    dmn = {
      source = "KamranKhan-Dev/dmn"
    }
  }
}

output "subnet" {
  value = provider::dmn::evaluate(file("networks.dmn"), "Network Lookup", {
    business_unit = "payments"
    environment   = "prod"
  })[0].subnet_id
}
```

### Signature

```text
evaluate(definitions string, decision_name string, inputs dynamic) dynamic
```

- `definitions` — the DMN XML, typically via `file("decision.dmn")`.
- `decision_name` — the `name` (or `id`) of the decision to evaluate.
- `inputs` — an object of input variables keyed by the decision's input names.

### Return shape

The function **always returns a list** of result entries, so callers index `[0]` for single-result tables. Each entry is an object of named outputs (or the bare value for a single unnamed output). Single-result hit policies return a 0- or 1-element list; multi-result policies return N entries; COLLECT aggregations return a single-element list with the scalar value. No match returns an empty list.

## Supported DMN

- DMN **1.2 – 1.5** namespaces (decision-table structure and hit-policy semantics are stable across these versions).
- Decision logic: **decision tables** and **literal expressions**. Boxed expressions (context, invocation, BKM, list, relation, conditional, iterator) return a clear "unsupported" error.
- Full **decision requirements graph (DRG)** resolution for the target decision's dependencies, with cycle detection.

FEEL expression evaluation is provided by the MIT-licensed [`pbinitiative/feel`](https://github.com/pbinitiative/feel), wrapped behind an internal interface; decision-table unary-test semantics are implemented in this provider.

## Development

```bash
make build    # go build ./...
make test     # unit tests
make testacc  # TF_ACC=1 acceptance tests (requires a Terraform >= 1.8 binary)
make fmt vet  # format and vet
```

## License

MIT. See [LICENSE](LICENSE).
