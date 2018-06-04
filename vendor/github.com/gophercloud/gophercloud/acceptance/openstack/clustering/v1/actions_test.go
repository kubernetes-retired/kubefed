// +build acceptance clustering actions

package v1

import (
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/clients"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/gophercloud/gophercloud/openstack/clustering/v1/actions"
	"github.com/gophercloud/gophercloud/pagination"
)

func TestActionsList(t *testing.T) {
	client, err := clients.NewClusteringV1Client()
	if err != nil {
		t.Fatalf("Unable to create a clustering client: %v", err)
	}

	opts := actions.ListOpts{
		Limit: 200,
	}

	err = actions.List(client, opts).EachPage(func(page pagination.Page) (bool, error) {
		actionInfos, err := actions.ExtractActions(page)
		if err != nil {
			return false, err
		}

		for _, actionInfo := range actionInfos {
			tools.PrintResource(t, actionInfo)
		}
		return true, nil
	})
}
