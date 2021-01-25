package kubego

import (
	"fmt"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/cmd/helm/search"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/strvals"
	"os"
	"path/filepath"
)

// Helm is a v3 helm client(wrapper)
type Helm struct {
	env  *cli.EnvSettings
	repo *repo.File
}

type HelmOpt func(settings *cli.EnvSettings)

// NewHelm creates a new v3 helm client(wrapper).
func NewHelm(opts ...HelmOpt) (*Helm, error) {
	h := &Helm{
		env:  cli.New(),
		repo: &repo.File{},
	}
	for _, o := range opts {
		o(h.env)
	}
	return h, nil
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

// Get gets a release by name
func (h *Helm) Get(namespace string, name string) (*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewGet(config)
	return client.Run(name)
}

// Upgrade upgrades a chart in the cluster
func (h *Helm) Upgrade(namespace string, name string, args map[string]string) (*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	upgrade := action.NewUpgrade(config)
	if upgrade.Version == "" {
		upgrade.Version = ">0.0.0-0"
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
	chrt, _, err := h.getLocalChart(name, &upgrade.ChartPathOptions)
	if err != nil {
		return nil, err
	}
	if req := chrt.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chrt, req); err != nil {
			return nil, err
		}
	}
	return upgrade.Run(name, chrt, vals)
}

// IsInstalled checks whether a release/chart is already installed on the cluster
func (h *Helm) IsInstalled(namespace string, release string) (bool, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return false, err
	}
	histClient := action.NewHistory(config)
	histClient.Max = 1
	if _, err := histClient.Run(release); err == driver.ErrReleaseNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// Install a chart/release in the given namespace
func (h *Helm) Install(namespace, name string, args map[string]string) (*release.Release, error) {
	installed, err := h.IsInstalled(namespace, name)
	if err != nil {
		return nil, err
	}
	if installed {
		return h.Upgrade(namespace, name, args)
	}
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewInstall(config)
	if client.Version == "" {
		client.Version = ">0.0.0-0"
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
	chrt, cp, err := h.getLocalChart(name, &client.ChartPathOptions)
	if err != nil {
		return nil, err
	}
	if req := chrt.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chrt, req); err != nil {
			man := &downloader.Manager{
				ChartPath:        cp,
				Keyring:          client.ChartPathOptions.Keyring,
				SkipUpdate:       false,
				Getters:          getters,
				RepositoryConfig: h.env.RepositoryConfig,
				RepositoryCache:  h.env.RepositoryCache,
			}
			if err := man.Update(); err != nil {
				return nil, err
			}
			return nil, err
		}
	}
	return client.Run(chrt, vals)
}

// Uninstall installs a chart by name
func (h *Helm) Uninstall(namespace, releaseName string) (*release.UninstallReleaseResponse, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewUninstall(config)
	return client.Run(releaseName)

}

// History returns a history of releases for the chart in the given namespace
func (h *Helm) History(namespace string, name string, max int) ([]*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	histClient := action.NewHistory(config)
	histClient.Max = max
	return histClient.Run(name)
}

// Rollback rolls back the chart by name to the previous version
func (h *Helm) Rollback(namespace string, name string) error {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return err
	}
	client := action.NewRollback(config)
	client.Recreate = true
	return client.Run(name)
}

// Status executes 'helm status' against the given release.
func (h *Helm) Status(namespace string, name string) (*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewStatus(config)
	client.ShowDescription = true
	return client.Run(name)
}

// List lists helm releases in the given namespace
func (h *Helm) ListReleases(namespace string) ([]*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	return action.NewList(config).Run()
}

// AddRepo adds or updates a helm repository
func (h *Helm) AddRepo(entry *repo.Entry) error {
	r, err := repo.NewChartRepository(entry, getter.All(h.env))
	if err != nil {
		return err
	}
	if _, err := r.DownloadIndexFile(); err != nil {
		return err
	}
	if h.repo.Has(entry.Name) {
		return nil
	}

	h.repo.Update(entry)
	return h.repo.WriteFile(h.env.RepositoryConfig, 0700)
}

// UpdateRepos updates all local helm repos
func (h *Helm) UpdateRepos() error {
	repoFile := h.env.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return err
	}
	for _, entry := range f.Repositories {
		r, err := repo.NewChartRepository(entry, getter.All(h.env))
		if err != nil {
			return err
		}
		if _, err := r.DownloadIndexFile(); err != nil {
			return err
		}
		if h.repo.Has(entry.Name) {
			return nil
		}

		h.repo.Update(entry)
	}
	return h.repo.WriteFile(h.env.RepositoryConfig, 0700)
}


// SearchCharts searches for a cached helm chart.
func (h *Helm) SearchCharts(term string, regex bool) ([]*search.Result, error) {
	repoFile := h.env.RepositoryConfig
	rf, err := repo.LoadFile(repoFile)
	if err != nil {
		return nil, err
	}
	i := search.NewIndex()
	for _, re := range rf.Repositories {
		f := filepath.Join(h.env.RepositoryCache, helmpath.CacheIndexFile(re.Name))
		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			if err := h.UpdateRepos(); err != nil {
				return nil, err
			}
			ind, _ = repo.LoadIndexFile(f)
		}
		if ind != nil {
			i.AddRepo(re.Name, ind, true)
		}
	}
	return i.Search(term, 25, regex)
}


// AllCharts returns all cached helm charts
func (h *Helm) AllCharts() ([]*search.Result, error) {
	repoFile := h.env.RepositoryConfig
	rf, err := repo.LoadFile(repoFile)
	if err != nil {
		return nil, err
	}
	i := search.NewIndex()
	for _, re := range rf.Repositories {
		f := filepath.Join(h.env.RepositoryCache, helmpath.CacheIndexFile(re.Name))
		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			if err := h.UpdateRepos(); err != nil {
				return nil, err
			}
			ind, _ = repo.LoadIndexFile(f)
		}
		if ind != nil {
			i.AddRepo(re.Name, ind, true)
		}
	}
	return i.All(), nil
}

func (c *Helm) getLocalChart(chartName string, chartPathOptions *action.ChartPathOptions) (*chart.Chart, string, error) {
	chartPath, err := chartPathOptions.LocateChart(chartName, c.env)
	if err != nil {
		return nil, "", err
	}
	helmChart, err := loader.Load(chartPath)
	if err != nil {
		return nil, "", err
	}

	if helmChart.Metadata.Deprecated {
		return nil, "", errors.New("deprecated chart")
	}
	return helmChart, chartPath, err
}
