// +build acceptance clustering autoscaling clusters profiles

package v1

import (
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/clients"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/gophercloud/gophercloud/openstack/clustering/v1/clusters"
	"github.com/gophercloud/gophercloud/openstack/clustering/v1/profiles"
	"github.com/gophercloud/gophercloud/pagination"
	th "github.com/gophercloud/gophercloud/testhelper"
)

var testName string

func TestAutoScaling(t *testing.T) {
	testName = tools.RandomString("TESTACC-", 8)
	profileCreate(t)
	profileGet(t)
	profileList(t)
	clusterCreate(t)
	clusterGet(t)
}

func profileCreate(t *testing.T) {
	client, err := clients.NewClusteringV1Client()
	if err != nil {
		t.Fatalf("Unable to create clustering client: %v", err)
	}

	networks := []map[string]interface{}{
		{"network": "sandbox-internal-net"},
	}

	props := map[string]interface{}{
		"name":            "centos-server",
		"flavor":          "t2.micro",
		"image":           "centos7.3-latest",
		"networks":        networks,
		"security_groups": "",
	}

	profileName := testName
	optsProfile := &profiles.CreateOpts{
		Metadata: map[string]interface{}{
			"foo":  "bar",
			"test": "123",
		},
		Name: profileName,
		Spec: profiles.Spec{
			Type:       "os.nova.server",
			Version:    "1.0",
			Properties: props,
		},
	}

	createResult := profiles.Create(client, optsProfile)
	th.AssertNoErr(t, createResult.Err)

	requestID := createResult.Header.Get("X-OpenStack-Request-Id")
	th.AssertEquals(t, true, requestID != "")

	profile, err := createResult.Extract()
	if err != nil {
		t.Fatalf("Unable to create profile %s: %v", profileName, err)
	} else {
		t.Logf("Profile created %v", profile)
	}

	th.AssertEquals(t, profileName, profile.Name)
	th.AssertEquals(t, "os.nova.server", profile.Spec.Type)
	th.AssertEquals(t, "1.0", profile.Spec.Version)
}

func profileGet(t *testing.T) {
	client, err := clients.NewClusteringV1Client()
	if err != nil {
		t.Fatalf("Unable to create clustering client: %v", err)
	}

	profileName := testName
	profile, err := profiles.Get(client, profileName).Extract()
	th.AssertNoErr(t, err)
	th.AssertEquals(t, profileName, profile.Name)

	tools.PrintResource(t, profile)
}

func profileList(t *testing.T) {
	client, err := clients.NewClusteringV1Client()
	if err != nil {
		t.Fatalf("Unable to create clustering client: %v", err)
	}

	testProfileFound := false
	profiles.List(client, profiles.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		allProfiles, err := profiles.ExtractProfiles(page)
		if err != nil {
			t.Fatalf("Error extracting page of profiles: %v", err)
		}

		for _, profile := range allProfiles {
			tools.PrintResource(t, profile)
			if profile.Name == testName {
				testProfileFound = true
				break
			}
		}

		empty, err := page.IsEmpty()

		th.AssertNoErr(t, err)

		// Expect the page IS NOT empty
		th.AssertEquals(t, false, empty)

		return true, nil
	})

	th.AssertEquals(t, true, testProfileFound)
}

func clusterCreate(t *testing.T) {
	client, err := clients.NewClusteringV1Client()
	if err != nil {
		t.Fatalf("Unable to create clustering client: %v", err)
	}

	clusterName := testName
	optsCluster := clusters.CreateOpts{
		Name:            clusterName,
		DesiredCapacity: 3,
		ProfileID:       testName,
		MinSize:         new(int),
		MaxSize:         20,
		Timeout:         3600,
		Metadata: map[string]interface{}{
			"foo": "bar",
			"test": map[string]interface{}{
				"nil_interface": interface{}(nil),
				"float_value":   float64(123.3),
				"string_value":  "test_string",
				"bool_value":    false,
			},
		},
		Config: map[string]interface{}{},
	}

	createResult := clusters.Create(client, optsCluster)
	th.AssertNoErr(t, createResult.Err)

	requestID := createResult.Header.Get("X-OpenStack-Request-Id")
	th.AssertEquals(t, true, requestID != "")

	location := createResult.Header.Get("Location")
	th.AssertEquals(t, true, location != "")

	actionID := ""
	locationFields := strings.Split(location, "actions/")
	if len(locationFields) >= 2 {
		actionID = locationFields[1]
	}
	th.AssertEquals(t, true, actionID != "")
	t.Logf("Cluster create action id: %s", actionID)

	cluster, err := createResult.Extract()
	if err != nil {
		t.Fatalf("Unable to create cluster %s: %v", clusterName, err)
	} else {
		t.Logf("Cluster created %+v", cluster)
	}

	th.AssertEquals(t, optsCluster.Name, cluster.Name)
	th.AssertEquals(t, optsCluster.DesiredCapacity, cluster.DesiredCapacity)
	th.AssertEquals(t, optsCluster.ProfileID, cluster.ProfileName)
	th.AssertEquals(t, *optsCluster.MinSize, cluster.MinSize)
	th.AssertEquals(t, optsCluster.MaxSize, cluster.MaxSize)
	th.AssertEquals(t, optsCluster.Timeout, cluster.Timeout)
	th.CheckDeepEquals(t, optsCluster.Metadata, cluster.Metadata)
	th.CheckDeepEquals(t, optsCluster.Config, cluster.Config)
}

func clusterGet(t *testing.T) {
	client, err := clients.NewClusteringV1Client()
	if err != nil {
		t.Fatalf("Unable to create clustering client: %v", err)
	}

	clusterName := testName
	cluster, err := clusters.Get(client, clusterName).Extract()
	th.AssertNoErr(t, err)
	th.AssertEquals(t, clusterName, cluster.Name)

	tools.PrintResource(t, cluster)
}
