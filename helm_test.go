package kubego_test

import (
	"github.com/autom8ter/kubego"
	"helm.sh/helm/v3/pkg/repo"
	"testing"
)

func TestHelm(t *testing.T) {
	h, err := kubego.NewHelm()
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := h.AddRepo(&repo.Entry{
		Name: "stable",
		URL:  "https://charts.helm.sh/stable",
	}); err != nil {
		t.Fatal(err)
	}
	releases, err := h.ListReleases("hermes-admin")
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, r := range releases {
		t.Log(r.Name)
	}
}
