// +build acceptance clustering policies

package v1

import (
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/clients"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/gophercloud/gophercloud/openstack/clustering/v1/policies"
	th "github.com/gophercloud/gophercloud/testhelper"
)

func TestPolicyList(t *testing.T) {
	client, err := clients.NewClusteringV1Client()
	th.AssertNoErr(t, err)

	allPages, err := policies.List(client, nil).AllPages()
	th.AssertNoErr(t, err)

	allPolicies, err := policies.ExtractPolicies(allPages)
	th.AssertNoErr(t, err)

	for _, v := range allPolicies {
		tools.PrintResource(t, v)

		if v.CreatedAt.IsZero() {
			t.Fatalf("CreatedAt value should not be zero")
		}
		t.Log("Created at: " + v.CreatedAt.String())

		if !v.UpdatedAt.IsZero() {
			t.Log("Updated at: " + v.UpdatedAt.String())
		}
	}
}

func TestPolicyCreateUpdateValidateDelete(t *testing.T) {
	client, err := clients.NewClusteringV1Client()
	th.AssertNoErr(t, err)
	testName := tools.RandomString("TESTACC-", 8)
	client.Microversion = "1.5"

	createOpts := policies.CreateOpts{
		Name: testName,
		Spec: policies.Spec{
			Description: "new policy description",
			Properties: map[string]interface{}{
				"destroy_after_deletion":  true,
				"grace_period":            60,
				"reduce_desired_capacity": false,
				"criteria":                "OLDEST_FIRST",
			},
			Type:    "senlin.policy.deletion",
			Version: "1.1",
		},
	}

	createResult := policies.Create(client, createOpts)
	th.AssertNoErr(t, createResult.Err)

	requestID := createResult.Header.Get("X-Openstack-Request-ID")
	th.AssertEquals(t, true, requestID != "")

	createdPolicy, err := createResult.Extract()
	th.AssertNoErr(t, err)

	defer policies.Delete(client, createdPolicy.ID)

	tools.PrintResource(t, createdPolicy)

	if createdPolicy.CreatedAt.IsZero() {
		t.Fatalf("CreatePolicy's CreatedAt value should not be zero")
	}
	t.Log("CreatePolicy created at: " + createdPolicy.CreatedAt.String())

	if !createdPolicy.UpdatedAt.IsZero() {
		t.Log("CreatePolicy updated at: " + createdPolicy.UpdatedAt.String())
	}

	updateOpts := policies.UpdateOpts{
		Name: testName + "-UPDATE",
	}

	updatePolicy, err := policies.Update(client, createdPolicy.ID, updateOpts).Extract()
	th.AssertNoErr(t, err)

	tools.PrintResource(t, updatePolicy)

	if updatePolicy.CreatedAt.IsZero() {
		t.Fatalf("UpdatePolicy's CreatedAt value should not be zero")
	}
	t.Log("UpdatePolicy created at: " + updatePolicy.CreatedAt.String())

	if !updatePolicy.UpdatedAt.IsZero() {
		t.Log("UpdatePolicy updated at: " + updatePolicy.UpdatedAt.String())
	}

	validateOpts := policies.ValidateOpts{
		Spec: createOpts.Spec,
	}

	validatePolicy, err := policies.Validate(client, validateOpts).Extract()
	th.AssertNoErr(t, err)

	tools.PrintResource(t, validatePolicy)

	if validatePolicy.Name != "validated_policy" {
		t.Fatalf("ValidatePolicy's Name value should be 'validated_policy'")
	}

	if validatePolicy.Spec.Version != createOpts.Spec.Version {
		t.Fatalf("ValidatePolicy's Version value should be ", createOpts.Spec.Version)
	}
}
