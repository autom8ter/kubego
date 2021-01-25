package kubego

import (
	"fmt"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/strvals"
	"io"
	"os"
)

// Helm is a v3 helm client(wrapper)
type Helm struct {
	env *cli.EnvSettings
}

// NewHelm creates a new v3 helm client(wrapper). If env is nil, default settings will be applied from env helm env vars
func NewHelm(env *cli.EnvSettings) *Helm {
	if env == nil {
		env = cli.New()
	}
	return &Helm{
		env: cli.New(),
	}
}

func (h *Helm) actionConfig(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if namespace == "" {
		namespace = h.env.Namespace()
	}
	if err := actionConfig.Init(h.env.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, args ...interface{}) {
		fmt.Printf(format, args...)
	}); err != nil {
		return nil, err
	}
	return actionConfig, nil
}

// InstallChart installs the chart at the given chart path in the given namespace with the given arguments
func (h *Helm) InstallChart(namespace string, chartPath string, args map[string]string) (*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	install := action.NewInstall(config)
	cp, err := install.ChartPathOptions.LocateChart(chartPath, h.env)
	if err != nil {
		return nil, err
	}
	getters := getter.All(h.env)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(getters)
	if err != nil {
		return nil, err
	}
	// Add args
	if err := strvals.ParseInto(args["set"], vals); err != nil {
		return nil, err
	}
	charts, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}
	if req := charts.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(charts, req); err != nil {
			if install.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          install.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getter.All(h.env),
					RepositoryConfig: h.env.RepositoryConfig,
					RepositoryCache:  h.env.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}
	return install.Run(charts, vals)
}

// UninstallChart installs the chart by name
func (h *Helm) UninstallRelease(namespace, releaseName string) (*release.UninstallReleaseResponse, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	return action.NewUninstall(config).Run(releaseName)
}

// ListCharts lists helm charts in the given namespace
func (h *Helm) ListCharts(namespace string, w io.Writer) error {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return err
	}
	return action.NewChartList(config).Run(w)
}

// ListReleases lists helm releases in the given namespace
func (h *Helm) ListReleases(namespace string) ([]*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	return action.NewList(config).Run()
}

// AddRepo adds a helm repository
func (h *Helm) AddRepo(entry *repo.Entry) error {
	r, err := repo.NewChartRepository(entry, getter.All(h.env))
	if err != nil {
		return err
	}
	if _, err := r.DownloadIndexFile(); err != nil {
		return err
	}
	return nil
}

// UpdateRepos updates all local helm repos
func (h *Helm) UpdateRepos() error {
	repoFile := h.env.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return err
	}
	for _, r := range f.Repositories {
		if err := h.AddRepo(r); err != nil {
			return err
		}
	}
	return nil
}
