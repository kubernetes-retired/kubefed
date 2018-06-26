package testing

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/openstack/clustering/v1/actions"
	"github.com/gophercloud/gophercloud/pagination"
	th "github.com/gophercloud/gophercloud/testhelper"
	fake "github.com/gophercloud/gophercloud/testhelper/client"
)

func TestListActions(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/actions", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `
		{
			"actions": [
				{
					"action": "NODE_DELETE",
					"cause": "RPC Request",
					"created_at": "2015-11-04T05:21:41Z",
					"data": {},
					"depended_by": ["ef67fe80-6547-40f2-ba1b-83e950aa38df"],
					"depends_on": ["ef67fe80-6547-40f2-ba1b-83e950aa38df"],
					"end_time": 1425550000.0,
					"id": "edce3528-864f-41fb-8759-f4707925cc09",
					"inputs": {},
					"interval": -1,
					"name": "node_delete_f0de9b9c",
					"outputs": {},
					"owner": null,
					"project": "f1fe61dcda2f4618a14c10dc7abc214d",
					"start_time": 1425550000.0,
					"status": "SUCCEEDED",
					"status_reason": "Action completed successfully.",
					"target": "f0de9b9c-6d48-4a46-af21-2ca8607777fe",
					"timeout": 3600,
					"updated_at": "2016-11-04T05:21:41Z",
					"user": "8bcd2cdca7684c02afc9e4f2fc0f0c79"
				},
				{
					"action": "NODE_DELETE",
					"cause": "RPC Request",
					"created_at": null,
					"data": {},
					"depended_by": ["ef67fe80-6547-40f2-ba1b-83e950aa38df"],
					"depends_on": ["ef67fe80-6547-40f2-ba1b-83e950aa38df"],
					"end_time": 1425550000.0,
					"id": "edce3528-864f-41fb-8759-f4707925cc09",
					"inputs": {},
					"interval": -1,
					"name": "node_delete_f0de9b9c",
					"outputs": {},
					"owner": null,
					"project": "f1fe61dcda2f4618a14c10dc7abc214d",
					"start_time": 1425550000.0,
					"status": "SUCCEEDED",
					"status_reason": "Action completed successfully.",
					"target": "f0de9b9c-6d48-4a46-af21-2ca8607777fe",
					"timeout": 3600,
					"updated_at": "",
					"user": "8bcd2cdca7684c02afc9e4f2fc0f0c79"
				}
			]
		}`)
	})

	pageCount := 0
	actions.List(fake.ServiceClient(), actions.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		pageCount++
		actual, err := actions.ExtractActions(page)
		if err != nil {
			t.Errorf("Failed to extract actions: %v", err)
			return false, err
		}

		createdAt, _ := time.Parse(time.RFC3339, "2015-11-04T05:21:41Z")
		updatedAt, _ := time.Parse(time.RFC3339, "2016-11-04T05:21:41Z")
		expected := []actions.Action{
			{
				Action:       "NODE_DELETE",
				Cause:        "RPC Request",
				CreatedAt:    createdAt,
				Data:         map[string]interface{}{},
				DependedBy:   []string{"ef67fe80-6547-40f2-ba1b-83e950aa38df"},
				DependsOn:    []string{"ef67fe80-6547-40f2-ba1b-83e950aa38df"},
				EndTime:      1425550000.0,
				ID:           "edce3528-864f-41fb-8759-f4707925cc09",
				Inputs:       make(map[string]interface{}),
				Interval:     -1,
				Name:         "node_delete_f0de9b9c",
				Outputs:      make(map[string]interface{}),
				Owner:        "",
				Project:      "f1fe61dcda2f4618a14c10dc7abc214d",
				StartTime:    1425550000.0,
				Status:       "SUCCEEDED",
				StatusReason: "Action completed successfully.",
				Target:       "f0de9b9c-6d48-4a46-af21-2ca8607777fe",
				Timeout:      3600,
				UpdatedAt:    updatedAt,
				User:         "8bcd2cdca7684c02afc9e4f2fc0f0c79",
			},
			{
				Action:       "NODE_DELETE",
				Cause:        "RPC Request",
				CreatedAt:    time.Time{},
				Data:         map[string]interface{}{},
				DependedBy:   []string{"ef67fe80-6547-40f2-ba1b-83e950aa38df"},
				DependsOn:    []string{"ef67fe80-6547-40f2-ba1b-83e950aa38df"},
				EndTime:      1425550000.0,
				ID:           "edce3528-864f-41fb-8759-f4707925cc09",
				Inputs:       make(map[string]interface{}),
				Interval:     -1,
				Name:         "node_delete_f0de9b9c",
				Outputs:      make(map[string]interface{}),
				Owner:        "",
				Project:      "f1fe61dcda2f4618a14c10dc7abc214d",
				StartTime:    1425550000.0,
				Status:       "SUCCEEDED",
				StatusReason: "Action completed successfully.",
				Target:       "f0de9b9c-6d48-4a46-af21-2ca8607777fe",
				Timeout:      3600,
				UpdatedAt:    time.Time{},
				User:         "8bcd2cdca7684c02afc9e4f2fc0f0c79",
			},
		}

		th.AssertDeepEquals(t, expected, actual)

		return true, nil
	})
	if pageCount != 1 {
		t.Errorf("Expected 1 page, got %d", pageCount)
	}
}

func TestGetAction(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/actions/edce3528-864f-41fb-8759-f4707925cc09", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
		{
			"action": {
				"action": "NODE_DELETE",
				"cause": "RPC Request",
				"created_at": "2015-11-04T05:21:41Z",
				"data": {},
				"depended_by": ["ef67fe80-6547-40f2-ba1b-83e950aa38df"],
				"depends_on": ["ef67fe80-6547-40f2-ba1b-83e950aa38df"],
				"end_time": 1425550000.0,
				"id": "edce3528-864f-41fb-8759-f4707925cc09",
				"inputs": {},
				"interval": -1,
				"name": "node_delete_f0de9b9c",
				"outputs": {},
				"owner": null,
				"project": "f1fe61dcda2f4618a14c10dc7abc214d",
				"start_time": 1425550000.0,
				"status": "SUCCEEDED",
				"status_reason": "Action completed successfully.",
				"target": "f0de9b9c-6d48-4a46-af21-2ca8607777fe",
				"timeout": 3600,
				"updated_at": "2016-11-04T05:21:41Z",
				"user": "8bcd2cdca7684c02afc9e4f2fc0f0c79"
			}
		}`)
	})

	createdAt, _ := time.Parse(time.RFC3339, "2015-11-04T05:21:41Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2016-11-04T05:21:41Z")
	expected := actions.Action{
		Action:       "NODE_DELETE",
		Cause:        "RPC Request",
		CreatedAt:    createdAt,
		Data:         map[string]interface{}{},
		DependedBy:   []string{"ef67fe80-6547-40f2-ba1b-83e950aa38df"},
		DependsOn:    []string{"ef67fe80-6547-40f2-ba1b-83e950aa38df"},
		EndTime:      1425550000.0,
		ID:           "edce3528-864f-41fb-8759-f4707925cc09",
		Inputs:       make(map[string]interface{}),
		Interval:     -1,
		Name:         "node_delete_f0de9b9c",
		Outputs:      make(map[string]interface{}),
		Owner:        "",
		Project:      "f1fe61dcda2f4618a14c10dc7abc214d",
		StartTime:    1425550000.0,
		Status:       "SUCCEEDED",
		StatusReason: "Action completed successfully.",
		Target:       "f0de9b9c-6d48-4a46-af21-2ca8607777fe",
		Timeout:      3600,
		UpdatedAt:    updatedAt,
		User:         "8bcd2cdca7684c02afc9e4f2fc0f0c79",
	}

	actual, err := actions.Get(fake.ServiceClient(), "edce3528-864f-41fb-8759-f4707925cc09").Extract()
	if err != nil {
		t.Errorf("Failed retrieving action. %v", err)
	} else {
		th.AssertDeepEquals(t, expected, *actual)
	}
}
