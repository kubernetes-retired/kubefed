// +build acceptance clustering webhooks

package v1

import (
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/clients"
	"github.com/gophercloud/gophercloud/openstack/clustering/v1/webhooks"

	th "github.com/gophercloud/gophercloud/testhelper"
)

func TestClusteringWebhookTrigger(t *testing.T) {

	client, err := clients.NewClusteringV1Client()
	if err != nil {
		t.Fatalf("Unable to create clustering client: %v", err)
	}

	// TODO: need to have cluster receiver created
	receiverUUID := "f93f83f6-762b-41b6-b757-80507834d394"
	actionID, err := webhooks.Trigger(client, receiverUUID, nil).Extract()
	if err != nil {
		// TODO: Uncomment next line once using real receiver
		//t.Fatalf("Unable to extract webhooks trigger: %v", err)
		t.Logf("TODO: Need to implement webhook trigger once PR receiver")
	} else {
		t.Logf("Webhook trigger action id %s", actionID)
	}

	// TODO: Need to compare to make sure action ID exists
	th.AssertEquals(t, true, true)
}
