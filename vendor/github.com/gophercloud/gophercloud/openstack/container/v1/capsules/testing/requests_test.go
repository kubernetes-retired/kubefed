package testing

import (
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/container/v1/capsules"
	"github.com/gophercloud/gophercloud/pagination"
	th "github.com/gophercloud/gophercloud/testhelper"
	fakeclient "github.com/gophercloud/gophercloud/testhelper/client"
)

func TestGetCapsule(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	HandleCapsuleGetSuccessfully(t)

	actualCapsule, err := capsules.Get(fakeclient.ServiceClient(), "cc654059-1a77-47a3-bfcf-715bde5aad9e").Extract()

	th.AssertNoErr(t, err)

	uuid := "cc654059-1a77-47a3-bfcf-715bde5aad9e"
	status := "Running"
	id := 1
	userID := "d33b18c384574fd2a3299447aac285f0"
	projectID := "6b8ffef2a0ac42ee87887b9cc98bdf68"
	cpu := float64(1)
	memory := "1024M"
	metaName := "test"

	createdAt, _ := time.Parse(gophercloud.RFC3339ZNoT, "2018-01-12 09:37:25+00:00")
	updatedAt, _ := time.Parse(gophercloud.RFC3339ZNoT, "2018-01-12 09:37:26+00:00")
	links := []interface{}{
		map[string]interface{}{
			"href": "http://10.10.10.10/v1/capsules/cc654059-1a77-47a3-bfcf-715bde5aad9e",
			"rel":  "self",
		},
		map[string]interface{}{
			"href": "http://10.10.10.10/capsules/cc654059-1a77-47a3-bfcf-715bde5aad9e",
			"rel":  "bookmark",
		},
	}
	capsuleVersion := "beta"
	restartPolicy := "always"
	metaLabels := map[string]string{
		"web": "app",
	}
	containersUUIDs := []string{
		"1739e28a-d391-4fd9-93a5-3ba3f29a4c9b",
	}
	addresses := map[string][]capsules.Address{
		"b1295212-64e1-471d-aa01-25ff46f9818d": []capsules.Address{
			{
				PreserveOnDelete: false,
				Addr:             "172.24.4.11",
				Port:             "8439060f-381a-4386-a518-33d5a4058636",
				Version:          float64(4),
				SubnetID:         "4a2bcd64-93ad-4436-9f48-3a7f9b267e0a",
			},
		},
	}
	volumesInfo := map[string][]string{
		"67618d54-dd55-4f7e-91b3-39ffb3ba7f5f": []string{
			"1739e28a-d391-4fd9-93a5-3ba3f29a4c9b",
		},
	}
	host := "test-host"
	statusReason := "No reason"

	capsuleID := 1
	containerID := 1
	containerName := "test-demo-omicron-13"
	containerUUID := "1739e28a-d391-4fd9-93a5-3ba3f29a4c9b"
	containerImage := "test"
	labels := map[string]string{
		"foo": "bar",
	}
	meta := map[string]string{
		"key1": "value1",
	}
	workDir := "/root"
	disk := 0

	containerIDBackend := "5109ebe2ca595777e994416208bd681b561b25ce493c34a234a1b68457cb53fb"
	command := "testcmd"
	ports := []int{
		80,
	}
	securityGroups := []string{
		"default",
	}
	imagePullPolicy := "ifnotpresent"
	runTime := "runc"
	taskState := "Creating"
	hostName := "test-hostname"
	environment := map[string]string{
		"USER1": "test",
	}
	websocketToken := "2ba16a5a-552f-422f-b511-bd786102691f"
	websocketUrl := "ws://10.10.10.10/"
	containerStatusReason := "No reason"
	statusDetail := "Just created"
	imageDriver := "docker"
	interactive := true
	autoRemove := false
	autoHeal := false
	containerRestartPolicy := map[string]string{
		"MaximumRetryCount": "0",
		"Name":              "always",
	}

	container1 := capsules.Container{
		Addresses:       addresses,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		UUID:            containerUUID,
		ID:              containerID,
		UserID:          userID,
		ProjectID:       projectID,
		CPU:             cpu,
		Status:          status,
		Memory:          memory,
		Host:            host,
		ContainerID:     containerIDBackend,
		CapsuleID:       capsuleID,
		Name:            containerName,
		Image:           containerImage,
		Labels:          labels,
		Meta:            meta,
		WorkDir:         workDir,
		Disk:            disk,
		Command:         command,
		Ports:           ports,
		SecurityGroups:  securityGroups,
		ImagePullPolicy: imagePullPolicy,
		Runtime:         runTime,
		TaskState:       taskState,
		HostName:        hostName,
		Environment:     environment,
		WebsocketToken:  websocketToken,
		WebsocketUrl:    websocketUrl,
		StatusReason:    containerStatusReason,
		StatusDetail:    statusDetail,
		ImageDriver:     imageDriver,
		AutoHeal:        autoHeal,
		AutoRemove:      autoRemove,
		Interactive:     interactive,
		RestartPolicy:   containerRestartPolicy,
	}
	containers := []capsules.Container{
		container1,
	}

	expectedCapsule := capsules.Capsule{
		UUID:            uuid,
		ID:              id,
		UserID:          userID,
		ProjectID:       projectID,
		CPU:             cpu,
		Status:          status,
		Memory:          memory,
		MetaName:        metaName,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		Links:           links,
		CapsuleVersion:  capsuleVersion,
		RestartPolicy:   restartPolicy,
		MetaLabels:      metaLabels,
		ContainersUUIDs: containersUUIDs,
		Addresses:       addresses,
		VolumesInfo:     volumesInfo,
		StatusReason:    statusReason,
		Host:            host,
		Containers:      containers,
	}

	th.AssertDeepEquals(t, &expectedCapsule, actualCapsule)
}

func TestCreateCapsule(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()
	HandleCapsuleCreateSuccessfully(t)
	template := new(capsules.Template)
	template.Bin = []byte(`{
		"capsuleVersion": "beta",
		"kind": "capsule",
		"metadata": {
			"labels": {
				"app": "web",
				"app1": "web1"
			},
			"name": "template"
		},
		"restartPolicy": "Always",
		"spec": {
			"containers": [
				{
					"command": [
						"/bin/bash"
					],
					"env": {
						"ENV1": "/usr/local/bin",
						"ENV2": "/usr/bin"
					},
					"image": "ubuntu",
					"imagePullPolicy": "ifnotpresent",
					"ports": [
						{
							"containerPort": 80,
							"hostPort": 80,
							"name": "nginx-port",
							"protocol": "TCP"
						}
					],
					"resources": {
						"requests": {
							"cpu": 1,
							"memory": 1024
						}
					},
					"workDir": "/root"
				}
			]
		}
	}`)
	createOpts := capsules.CreateOpts{
		TemplateOpts: template,
	}
	err := capsules.Create(fakeclient.ServiceClient(), createOpts).ExtractErr()
	th.AssertNoErr(t, err)
}

func TestListCapsule(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	HandleCapsuleListSuccessfully(t)

	count := 0
	results := capsules.List(fakeclient.ServiceClient(), nil)
	err := results.EachPage(func(page pagination.Page) (bool, error) {
		count++
		actual, err := capsules.ExtractCapsules(page)
		if err != nil {
			t.Errorf("Failed to extract capsules: %v", err)
			return false, err
		}
		uuid := "cc654059-1a77-47a3-bfcf-715bde5aad9e"
		status := "Running"
		id := 1
		userID := "d33b18c384574fd2a3299447aac285f0"
		projectID := "6b8ffef2a0ac42ee87887b9cc98bdf68"
		cpu := float64(1)
		memory := "1024M"
		metaName := "test"

		createdAt, _ := time.Parse(gophercloud.RFC3339ZNoT, "2018-01-12 09:37:25+00:00")
		updatedAt, _ := time.Parse(gophercloud.RFC3339ZNoT, "2018-01-12 09:37:25+01:00")
		links := []interface{}{
			map[string]interface{}{
				"href": "http://10.10.10.10/v1/capsules/cc654059-1a77-47a3-bfcf-715bde5aad9e",
				"rel":  "self",
			},
			map[string]interface{}{
				"href": "http://10.10.10.10/capsules/cc654059-1a77-47a3-bfcf-715bde5aad9e",
				"rel":  "bookmark",
			},
		}
		capsuleVersion := "beta"
		restartPolicy := "always"
		metaLabels := map[string]string{
			"web": "app",
		}
		containersUUIDs := []string{
			"1739e28a-d391-4fd9-93a5-3ba3f29a4c9b",
			"d1469e8d-bcbc-43fc-b163-8b9b6a740930",
		}
		addresses := map[string][]capsules.Address{
			"b1295212-64e1-471d-aa01-25ff46f9818d": []capsules.Address{
				{
					PreserveOnDelete: false,
					Addr:             "172.24.4.11",
					Port:             "8439060f-381a-4386-a518-33d5a4058636",
					Version:          float64(4),
					SubnetID:         "4a2bcd64-93ad-4436-9f48-3a7f9b267e0a",
				},
			},
		}
		volumesInfo := map[string][]string{
			"67618d54-dd55-4f7e-91b3-39ffb3ba7f5f": []string{
				"4b725a92-2197-497b-b6b1-fb8caa4cb99b",
			},
		}
		host := "test-host"
		statusReason := "No reason"

		expected := []capsules.Capsule{
			{
				UUID:            uuid,
				ID:              id,
				UserID:          userID,
				ProjectID:       projectID,
				CPU:             cpu,
				Status:          status,
				Memory:          memory,
				MetaName:        metaName,
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
				Links:           links,
				CapsuleVersion:  capsuleVersion,
				RestartPolicy:   restartPolicy,
				MetaLabels:      metaLabels,
				ContainersUUIDs: containersUUIDs,
				Addresses:       addresses,
				VolumesInfo:     volumesInfo,
				StatusReason:    statusReason,
				Host:            host,
			},
		}
		th.CheckDeepEquals(t, expected, actual)

		return true, nil
	})
	th.AssertNoErr(t, err)

	if count != 1 {
		t.Errorf("Expected 1 page, got %d", count)
	}
}

func TestDelete(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	HandleCapsuleDeleteSuccessfully(t)

	res := capsules.Delete(fakeclient.ServiceClient(), "963a239d-3946-452b-be5a-055eab65a421")
	th.AssertNoErr(t, res.Err)
}
