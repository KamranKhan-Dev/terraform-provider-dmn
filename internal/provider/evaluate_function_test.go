package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccEvaluateFunction_basic(t *testing.T) {
	config := `
output "result" {
  value = provider::dmn::evaluate(file("testdata/simple.dmn"), "Network Lookup", {
    business_unit = "payments"
    environment   = "prod"
  })
}
`
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result",
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.ObjectExact(map[string]knownvalue.Check{
								"subnet_id": knownvalue.StringExact("subnet-aaa"),
							}),
						}),
					),
				},
			},
		},
	})
}

func TestAccEvaluateFunction_decisionNotFound(t *testing.T) {
	config := `
output "result" {
  value = provider::dmn::evaluate(file("testdata/simple.dmn"), "Nope", {})
}
`
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`not found`),
			},
		},
	})
}
