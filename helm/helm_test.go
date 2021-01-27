package helm_test

import (
	"github.com/autom8ter/kubego/helm"
	"testing"
)

func TestHelm(t *testing.T) {
	h, err := helm.NewHelm()
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := h.AddRepo(helm.StableCharts); err != nil {
		t.Fatal(err)
	}
	results, err := h.AllCharts()
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(results) == 0 {
		t.Fatal("failed to load stable charts")
	}
	for _, r := range results {
		t.Log(r.Name)
	}
	releases, err := h.SearchReleases("hermes-admin", "", 5, 0)
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, r := range releases {
		t.Log(r.Name)
	}

}
