package sprintly

import (
	"fmt"
	"net/http"
	"testing"
)

var testingDeploy = Deploy{
	Environment: "staging",
	Items: []Item{
		{
			Number: 188,
			Title:  "Who knows ...",
		},
	},
}

var testingDeployJson = `
{
	"environment": "staging",
	"items": [
		{
			"number": 188,
			"title": "Who knows ..."
		}
	]
}
`

var (
	testingDeploySlice     = []Deploy{testingDeploy}
	testingDeploySliceJson = fmt.Sprintf("[%v]", testingDeployJson)
)

func TestDeploys_List(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	mux.HandleFunc("/products/1/deploys.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "GET")
		fmt.Fprint(w, testingDeploySliceJson)
	})

	deploys, _, err := client.Deploys.List(1, nil)
	if err != nil {
		t.Errorf("Deploys.List failed: %v", err)
		return
	}

	ensureEqual(t, deploys, testingDeploySlice)
}

func TestDeploys_Create(t *testing.T) {
	client, server, mux := setup()
	defer server.Close()

	args := DeployCreateArgs{
		Environment: "staging",
		ItemNumbers: []int{1, 2, 3, 4, 5},
	}

	mux.HandleFunc("/products/1/deploys.json", func(w http.ResponseWriter, r *http.Request) {
		ensureMethod(t, r, "POST")

		var receivedArgs DeployCreateArgs
		if err := decodeArgs(&receivedArgs, r); err != nil {
			t.Error(err)
			return
		}

		ensureEqual(t, &receivedArgs, &args)
		fmt.Fprint(w, testingDeployJson)
	})

	deploy, _, err := client.Deploys.Create(1, &args)
	if err != nil {
		t.Errorf("Deploys.Create failed: %v", err)
		return
	}

	ensureEqual(t, deploy, &testingDeploy)
}
