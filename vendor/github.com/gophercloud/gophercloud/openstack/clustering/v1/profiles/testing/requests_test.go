package testing

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/openstack/clustering/v1/profiles"
	"github.com/gophercloud/gophercloud/pagination"
	th "github.com/gophercloud/gophercloud/testhelper"
	fake "github.com/gophercloud/gophercloud/testhelper/client"
)

func TestCreateProfile(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"profile": {
				"created_at": "2016-01-03T16:22:23Z",
				"domain": null,
				"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
				"metadata": {},
				"name": "test-profile",
				"project": "42d9e9663331431f97b75e25136307ff",
				"spec": {
					"properties": {
						"flavor": "t2.small",
						"image": "centos7.3-latest",
						"name": "centos_server",
						"networks": [
								{
									"network": "private-network"
								}
						]
					},
					"type": "os.nova.server",
					"version": "1.0"
				},
				"type": "os.nova.server-1.0",
				"updated_at": "2016-01-03T17:22:23Z",
				"user": "5e5bf8027826429c96af157f68dc9072"
			}
		}`)
	})

	networks := []map[string]interface{}{
		{"network": "private-network"},
	}

	props := map[string]interface{}{
		"name":            "test_gopher_cloud_profile",
		"flavor":          "t2.small",
		"image":           "centos7.3-latest",
		"networks":        networks,
		"security_groups": "",
	}

	optsProfile := &profiles.CreateOpts{
		Name: "TestProfile",
		Spec: profiles.Spec{
			Type:       "os.nova.server",
			Version:    "1.0",
			Properties: props,
		},
	}

	profile, err := profiles.Create(fake.ServiceClient(), optsProfile).Extract()
	if err != nil {
		t.Errorf("Failed to extract profile: %v", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, "2016-01-03T16:22:23Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2016-01-03T17:22:23Z")

	expected := profiles.Profile{
		CreatedAt: createdAt,
		Domain:    "",
		ID:        "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
		Metadata:  map[string]interface{}{},
		Name:      "test-profile",
		Project:   "42d9e9663331431f97b75e25136307ff",
		Spec: profiles.Spec{
			Properties: map[string]interface{}{
				"flavor": "t2.small",
				"image":  "centos7.3-latest",
				"name":   "centos_server",
				"networks": []interface{}{
					map[string]interface{}{"network": "private-network"},
				},
			},
			Type:    "os.nova.server",
			Version: "1.0",
		},
		Type:      "os.nova.server-1.0",
		UpdatedAt: updatedAt,
		User:      "5e5bf8027826429c96af157f68dc9072",
	}

	th.AssertDeepEquals(t, expected, *profile)
}

func TestCreateProfileInvalidTimeFloat(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"profile": {
				"created_at": 123456789.0,
				"domain": null,
				"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
				"metadata": {},
				"name": "test-profile",
				"project": "42d9e9663331431f97b75e25136307ff",
				"spec": {
					"properties": {
						"flavor": "t2.small",
						"image": "centos7.3-latest",
						"name": "centos_server",
						"networks": [
								{
									"network": "private-network"
								}
						]
					},
					"type": "os.nova.server",
					"version": "1.0"
				},
				"type": "os.nova.server-1.0",
				"updated_at": 123456789.0,
				"user": "5e5bf8027826429c96af157f68dc9072"
			}
		}`)
	})

	optsProfile := &profiles.CreateOpts{
		Name: "TestProfile",
		Spec: profiles.Spec{
			Type:       "os.nova.server",
			Version:    "1.0",
			Properties: map[string]interface{}{},
		},
	}

	_, err := profiles.Create(fake.ServiceClient(), optsProfile).Extract()
	th.AssertEquals(t, false, err == nil)
}

func TestCreateProfileInvalidTimeString(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"profile": {
				"created_at": "invalid_time",
				"domain": null,
				"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
				"metadata": {},
				"name": "test-profile",
				"project": "42d9e9663331431f97b75e25136307ff",
				"spec": {
					"properties": {
						"flavor": "t2.small",
						"image": "centos7.3-latest",
						"name": "centos_server",
						"networks": [
								{
									"network": "private-network"
								}
						]
					},
					"type": "os.nova.server",
					"version": "1.0"
				},
				"type": "os.nova.server-1.0",
				"updated_at": "invalid_time",
				"user": "5e5bf8027826429c96af157f68dc9072"
			}
		}`)
	})

	optsProfile := &profiles.CreateOpts{
		Name: "TestProfile",
		Spec: profiles.Spec{
			Type:       "os.nova.server",
			Version:    "1.0",
			Properties: map[string]interface{}{},
		},
	}

	_, err := profiles.Create(fake.ServiceClient(), optsProfile).Extract()
	th.AssertEquals(t, false, err == nil)
}

func TestGetProfile(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/profiles/9e1c6f42-acf5-4688-be2c-8ce954ef0f23", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"profile": {
				"created_at": "2016-01-03T16:22:23Z",
				"domain": null,
				"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
				"metadata": {},
				"name": "pserver",
				"project": "42d9e9663331431f97b75e25136307ff",
				"spec": {
					"properties": {
						"flavor": 1,
						"image": "cirros-0.3.4-x86_64-uec",
						"key_name": "oskey",
						"name": "cirros_server",
						"networks": [
							{
								"network": "private"
							}
						]
					},
					"type": "os.nova.server",
					"version": "1.0"
				},
				"type": "os.nova.server-1.0",
				"updated_at": null,
				"user": "5e5bf8027826429c96af157f68dc9072"
			}
		}`)
	})

	actual, err := profiles.Get(fake.ServiceClient(), "9e1c6f42-acf5-4688-be2c-8ce954ef0f23").Extract()
	if err != nil {
		t.Errorf("Failed to get profile. %v", err)
	} else {
		createdAt, _ := time.Parse(time.RFC3339, "2016-01-03T16:22:23Z")
		updatedAt := time.Time{}
		expected := profiles.Profile{
			CreatedAt: createdAt,
			Domain:    "",
			ID:        "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
			Metadata:  map[string]interface{}{},
			Name:      "pserver",
			Project:   "42d9e9663331431f97b75e25136307ff",
			Spec: profiles.Spec{
				Properties: map[string]interface{}{
					"flavor":   float64(1),
					"image":    "cirros-0.3.4-x86_64-uec",
					"key_name": "oskey",
					"name":     "cirros_server",
					"networks": []interface{}{
						map[string]interface{}{"network": "private"},
					},
				},
				Type:    "os.nova.server",
				Version: "1.0",
			},
			Type:      "os.nova.server-1.0",
			UpdatedAt: updatedAt,
			User:      "5e5bf8027826429c96af157f68dc9072",
		}

		th.AssertDeepEquals(t, expected, *actual)
	}
}

func TestGetProfileInvalidCreatedAtTime(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/profiles/9e1c6f42-acf5-4688-be2c-8ce954ef0f23", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"profile": {
				"created_at": 1234567890.0,
				"domain": null,
				"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
				"metadata": {},
				"name": "pserver",
				"project": "42d9e9663331431f97b75e25136307ff",
				"spec": {
					"properties": {
						"flavor": 1,
						"image": "cirros-0.3.4-x86_64-uec",
						"key_name": "oskey",
						"name": "cirros_server",
						"networks": [
							{
								"network": "private"
							}
						]
					},
					"type": "os.nova.server",
					"version": "1.0"
				},
				"type": "os.nova.server-1.0",
				"updated_at": "",
				"user": "5e5bf8027826429c96af157f68dc9072"
			}
		}`)
	})

	_, err := profiles.Get(fake.ServiceClient(), "9e1c6f42-acf5-4688-be2c-8ce954ef0f23").Extract()
	th.AssertEquals(t, false, err == nil)
}

func TestGetProfileInvalidUpdatedAtTime(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/profiles/9e1c6f42-acf5-4688-be2c-8ce954ef0f23", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"profile": {
				"created_at": null,
				"domain": null,
				"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
				"metadata": {},
				"name": "pserver",
				"project": "42d9e9663331431f97b75e25136307ff",
				"spec": {
					"properties": {
						"flavor": 1,
						"image": "cirros-0.3.4-x86_64-uec",
						"key_name": "oskey",
						"name": "cirros_server",
						"networks": [
							{
								"network": "private"
							}
						]
					},
					"type": "os.nova.server",
					"version": "1.0"
				},
				"type": "os.nova.server-1.0",
				"updated_at": 1234567890.0,
				"user": "5e5bf8027826429c96af157f68dc9072"
			}
		}`)
	})

	_, err := profiles.Get(fake.ServiceClient(), "9e1c6f42-acf5-4688-be2c-8ce954ef0f23").Extract()
	th.AssertEquals(t, false, err == nil)
}
func TestListProfiles(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"profiles": [
				{
					"created_at": "2016-01-03T16:22:23Z",
					"domain": null,
					"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
					"metadata": {},
					"name": "pserver",
					"project": "42d9e9663331431f97b75e25136307ff",
					"spec": {
						"properties": {
							"flavor": "t2.small",
							"image": "cirros-0.3.4-x86_64-uec",
							"key_name": "oskey",
							"name": "cirros_server",
							"networks": [
								{
									"network": "private"
								}
							]
						},
						"type": "os.nova.server",
						"version": 1.0
					},
					"type": "os.nova.server-1.0",
					"updated_at": "2016-01-03T17:22:23Z",
					"user": "5e5bf8027826429c96af157f68dc9072"
				},
				{
					"created_at": null,
					"domain": null,
					"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
					"metadata": {},
					"name": "pserver",
					"project": "42d9e9663331431f97b75e25136307ff",
					"spec": {
						"properties": {
							"flavor": "t2.small",
							"image": "cirros-0.3.4-x86_64-uec",
							"key_name": "oskey",
							"name": "cirros_server",
							"networks": [
								{
									"network": "private"
								}
							]
						},
						"type": "os.nova.server",
						"version": 1.0
					},
					"type": "os.nova.server-1.0",
					"updated_at": null,
					"user": "5e5bf8027826429c96af157f68dc9072"
				},
				{
					"created_at": "",
					"domain": null,
					"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
					"metadata": {},
					"name": "pserver",
					"project": "42d9e9663331431f97b75e25136307ff",
					"spec": {
						"properties": {
							"flavor": "t2.small",
							"image": "cirros-0.3.4-x86_64-uec",
							"key_name": "oskey",
							"name": "cirros_server",
							"networks": [
								{
									"network": "private"
								}
							]
						},
						"type": "os.nova.server",
						"version": "1.0"
					},
					"type": "os.nova.server-1.0",
					"updated_at": "",
					"user": "5e5bf8027826429c96af157f68dc9072"
				}
		    ]
		}`)
	})

	count := 0
	profiles.List(fake.ServiceClient(), profiles.ListOpts{GlobalProject: new(bool)}).EachPage(func(page pagination.Page) (bool, error) {
		count++
		actual, err := profiles.ExtractProfiles(page)
		if err != nil {
			t.Errorf("Failed to extract profiles: %v", err)
			return false, err
		}

		createdAt, _ := time.Parse(time.RFC3339, "2016-01-03T16:22:23Z")
		updatedAt, _ := time.Parse(time.RFC3339, "2016-01-03T17:22:23Z")

		expected := []profiles.Profile{
			{
				CreatedAt: createdAt,
				Domain:    "",
				ID:        "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
				Metadata:  map[string]interface{}{},
				Name:      "pserver",
				Project:   "42d9e9663331431f97b75e25136307ff",
				Spec: profiles.Spec{
					Properties: map[string]interface{}{
						"flavor":   "t2.small",
						"image":    "cirros-0.3.4-x86_64-uec",
						"key_name": "oskey",
						"name":     "cirros_server",
						"networks": []interface{}{
							map[string]interface{}{"network": "private"},
						},
					},
					Type:    "os.nova.server",
					Version: "1.0",
				},
				Type:      "os.nova.server-1.0",
				UpdatedAt: updatedAt,
				User:      "5e5bf8027826429c96af157f68dc9072",
			},
			{
				CreatedAt: time.Time{},
				Domain:    "",
				ID:        "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
				Metadata:  map[string]interface{}{},
				Name:      "pserver",
				Project:   "42d9e9663331431f97b75e25136307ff",
				Spec: profiles.Spec{
					Properties: map[string]interface{}{
						"flavor":   "t2.small",
						"image":    "cirros-0.3.4-x86_64-uec",
						"key_name": "oskey",
						"name":     "cirros_server",
						"networks": []interface{}{
							map[string]interface{}{"network": "private"},
						},
					},
					Type:    "os.nova.server",
					Version: "1.0",
				},
				Type:      "os.nova.server-1.0",
				UpdatedAt: time.Time{},
				User:      "5e5bf8027826429c96af157f68dc9072",
			},
			{
				CreatedAt: time.Time{},
				Domain:    "",
				ID:        "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
				Metadata:  map[string]interface{}{},
				Name:      "pserver",
				Project:   "42d9e9663331431f97b75e25136307ff",
				Spec: profiles.Spec{
					Properties: map[string]interface{}{
						"flavor":   "t2.small",
						"image":    "cirros-0.3.4-x86_64-uec",
						"key_name": "oskey",
						"name":     "cirros_server",
						"networks": []interface{}{
							map[string]interface{}{"network": "private"},
						},
					},
					Type:    "os.nova.server",
					Version: "1.0",
				},
				Type:      "os.nova.server-1.0",
				UpdatedAt: time.Time{},
				User:      "5e5bf8027826429c96af157f68dc9072",
			},
		}

		th.AssertDeepEquals(t, expected, actual)

		return true, nil
	})

	if count != 1 {
		t.Errorf("Expected 1 page of profiles, got %d pages instead", count)
	}
}

func TestListProfilesInvalidTimeFloat(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"profiles": [
				{
					"created_at": 123456789.0,
					"domain": null,
					"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
					"metadata": {},
					"name": "pserver",
					"project": "42d9e9663331431f97b75e25136307ff",
					"spec": {
						"properties": {
							"flavor": 1,
							"image": "cirros-0.3.4-x86_64-uec",
							"key_name": "oskey",
							"name": "cirros_server",
							"networks": [
								{
									"network": "private"
								}
							]
						},
						"type": "os.nova.server",
						"version": 1.0
					},
					"type": "os.nova.server-1.0",
					"updated_at": 123456789.0,
					"user": "5e5bf8027826429c96af157f68dc9072"
				}
		    ]
		}`)
	})

	count := 0
	err := profiles.List(fake.ServiceClient(), profiles.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		count++
		return true, nil
	})

	th.AssertEquals(t, false, err == nil)
	if count != 0 {
		t.Errorf("Expected 0 page of profiles, got %d pages instead", count)
	}
}

func TestListProfilesInvalidTimeString(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"profiles": [
				{
					"created_at": "invalid",
					"domain": null,
					"id": "9e1c6f42-acf5-4688-be2c-8ce954ef0f23",
					"metadata": {},
					"name": "pserver",
					"project": "42d9e9663331431f97b75e25136307ff",
					"spec": {
						"properties": {
							"flavor": 1,
							"image": "cirros-0.3.4-x86_64-uec",
							"key_name": "oskey",
							"name": "cirros_server",
							"networks": [
								{
									"network": "private"
								}
							]
						},
						"type": "os.nova.server",
						"version": 1.0
					},
					"type": "os.nova.server-1.0",
					"updated_at": "invalid",
					"user": "5e5bf8027826429c96af157f68dc9072"
				}
		    ]
		}`)
	})

	count := 0
	err := profiles.List(fake.ServiceClient(), profiles.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		count++
		return true, nil
	})

	th.AssertEquals(t, false, err == nil)
	if count != 0 {
		t.Errorf("Expected 0 page of profiles, got %d pages instead", count)
	}
}
