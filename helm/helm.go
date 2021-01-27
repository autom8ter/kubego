package helm

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
	"os"
	"path/filepath"
)

// StableCharts is a helm repo entry for the standard stable helm charts: https://charts.helm.sh/stable
var StableCharts = &repo.Entry{
	Name: "stable",
	URL:  "https://charts.helm.sh/stable",
}

// Helm is a v3 helm client(wrapper)
type Helm struct {
	env    *cli.EnvSettings
	repo   *repo.File
	logger func(format string, args ...interface{})
}

// HelmOpt is an optional argument to modify the helm client
type HelmOpt func(h *Helm)

// WithLogger modifies default logger
func WithLogger(logger func(format string, args ...interface{})) HelmOpt {
	return func(h *Helm) {
		h.logger = logger
	}
}

// WithEnvFunc modifies helm environmental settings
func WithEnvFunc(fn func(settings *cli.EnvSettings)) HelmOpt {
	return func(h *Helm) {
		fn(h.env)
	}
}

// NewHelm creates a new v3 helm client(wrapper).
func NewHelm(opts ...HelmOpt) (*Helm, error) {
	h := &Helm{
		env:  cli.New(),
		repo: &repo.File{},
		logger: func(format string, args ...interface{}) {
			fmt.Printf(format, args...)
		},
	}
	for _, o := range opts {
		o(h)
	}
	return h, nil
}

func (h *Helm) actionConfig(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if namespace == "" {
		namespace = h.env.Namespace()
	}
	if err := actionConfig.Init(h.env.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), h.logger); err != nil {
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
func (h *Helm) Upgrade(namespace string, chartName, releaseName string, recreate bool, configVals map[string]string) (*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	upgrade := action.NewUpgrade(config)
	if upgrade.Version == "" {
		upgrade.Version = ">0.0.0-0"
	}
	upgrade.Namespace = namespace
	upgrade.Recreate = recreate
	upgrade.Wait = true
	getters := getter.All(h.env)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(getters)
	if err != nil {
		return nil, err
	}
	for k, v := range configVals {
		vals[k] = v
	}
	chrt, _, err := h.getLocalChart(chartName, &upgrade.ChartPathOptions)
	if err != nil {
		return nil, err
	}
	if req := chrt.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chrt, req); err != nil {
			return nil, err
		}
	}
	return upgrade.Run(releaseName, chrt, vals)
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
func (h *Helm) Install(namespace, chartName, releaseName string, createNamespace bool, configVals map[string]string) (*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewInstall(config)
	if client.Version == "" {
		client.Version = ">0.0.0-0"
	}
	client.Namespace = namespace
	client.CreateNamespace = createNamespace
	client.IncludeCRDs = true
	client.Wait = true
	client.ReleaseName = releaseName
	getters := getter.All(h.env)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(getters)
	if err != nil {
		return nil, err
	}
	for k, v := range configVals {
		vals[k] = v
	}
	chrt, cp, err := h.getLocalChart(chartName, &client.ChartPathOptions)
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

// Uninstall uninstalls a release by name in the given namespace
func (h *Helm) Uninstall(namespace, releaseName string) (*release.UninstallReleaseResponse, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewUninstall(config)
	return client.Run(releaseName)

}

// History returns a history for the named release in the given namespace
func (h *Helm) History(namespace string, release string, max int) ([]*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	histClient := action.NewHistory(config)
	histClient.Max = max
	return histClient.Run(release)
}

// Rollback rolls back the chart by name to the previous version
func (h *Helm) Rollback(namespace string, release string) error {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return err
	}
	client := action.NewRollback(config)
	client.Recreate = true
	return client.Run(release)
}

// Status executes 'helm status' against the given release.
func (h *Helm) Status(namespace string, release string) (*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewStatus(config)
	client.ShowDescription = true
	return client.Run(release)
}

// SearchReleases searches for helm releases. If namespace is empty, all namespaces will be searched.
func (h *Helm) SearchReleases(namespace, selector string, limit, offset int) ([]*release.Release, error) {
	config, err := h.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewList(config)
	client.StateMask = action.ListAll
	client.Limit = limit
	client.Selector = selector
	client.AllNamespaces = namespace == ""
	client.Offset = offset
	return client.Run()
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
