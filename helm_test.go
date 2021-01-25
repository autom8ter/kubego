package kubego_test

import (
	"github.com/autom8ter/kubego"
	"helm.sh/helm/v3/pkg/repo"
	"testing"
)

func TestHelm(t *testing.T) {
	h := kubego.NewHelm(nil)
	if err := h.AddRepo(&repo.Entry{
		Name: "stable",
		URL:  "https://charts.helm.sh/stable",
	}); err != nil {
		t.Fatal(err)
	}
}
