package cluster

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/test-clusters/testclusters-go/pkg/cluster/health"
	"math/rand"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/config"
	configTypes "github.com/k3d-io/k3d/v5/pkg/config/types"
	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3dTypes "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/phayes/freeport"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/retry"

	"github.com/test-clusters/testclusters-go/pkg/naming"
)

// make this configurable so devs can deploy different clusters or have the need for a custom container naming?
const appName = "k8s-containers"
const DefaultNamespace = "default"

// k3s versions
// warning: k3s versions are tagged with a `+` separator before `k3s1`, but k3s images use `-`.
const (
	K3sVersion1_26 = "v1.26.2-k3s1"
	K3sVersion1_28 = "v1.28.2-k3s1"
)

// defaultPrefix contains a marker that will be prepended to testcluster-go containers to distinguish these from other
// containers (possibly also made with k3d).
const defaultPrefix = "tc-"

// restrictedContainerNameChars allows only for a prefix lower double-digit number of characters as there will be
// added further pre- and suffixes:
//
//   - "k3d-" by k3d itself
//   - the defaultPrefix
//   - 8 alphanumeric characters to identify containers of the same cluster
//   - the pod name
const restrictedContainerNameChars = `[a-zA-Z0-9][a-zA-Z0-9_.-]{1..37}`

var restrictedContainerNamePattern = regexp.MustCompile(`^` + restrictedContainerNameChars + `$`)

const (
	nodeHealthEvictionHardEmpty = ""
)

// Loglevel describes the verbosity of log messages being printed from testcluster-go
type Loglevel int

const (
	Error Loglevel = iota
	Warning
	Info
	Debug
)

type testcluster interface {
	// Terminate stops all workloads and reclaims spent computational resources.
	Terminate(ctx context.Context) error
}

// Opts allows customizing the K3d cluster's creation and operation.
type Opts struct {
	// ClusterNamePrefix will be used to name the cluster so developers can discover
	// and address a cluster among others.
	// The cluster name prefix will be delimited by the default delimiter "-".
	// Defaults to the empty string which is allowed.
	ClusterNamePrefix string
	// LogLevel allows increases or decreases the testcluster-go's log verbosity.
	// Currently, k3d's log verbosity cannot be adjusted with this field.
	// Defaults to Warning.
	LogLevel Loglevel
	// NodeConditionEvictionHardArg allows tuning the kubelet's evictionHard argument.
	// Defaults to empty string.
	// See health.KubeletEvictionFsByPercentage for a helper function
	NodeConditionEvictionHardArg string
}

// K3dCluster abstracts the cluster management during developer tests.
type K3dCluster struct {
	containerRuntime    runtimes.Runtime
	clusterConfig       *v1alpha5.ClusterConfig
	kubeConfig          *api.Config
	clientConfig        *rest.Config
	ClusterName         string
	AdminServiceAccount string
	clientSet           kubernetes.Interface
}

// NewK3dCluster creates a completely new cluster within the provided container
// engine and default values. This method is the usual entry point of a test with
// testclusters-go.
func NewK3dCluster(t *testing.T) *K3dCluster {
	defaultOpts := Opts{
		ClusterNamePrefix:            "",
		LogLevel:                     Warning,
		NodeConditionEvictionHardArg: nodeHealthEvictionHardEmpty,
	}
	return NewK3dClusterWithOpts(t, defaultOpts)
}

// NewK3dClusterWithOpts creates like NewK3dCluster a new cluster but with more control over customization.
func NewK3dClusterWithOpts(t *testing.T, opts Opts) *K3dCluster {
	cluster := mustSetupCluster(t, opts)
	registerTearDown(t, cluster)

	return cluster
}

func mustSetupCluster(t *testing.T, opts Opts) *K3dCluster {
	l.Log().Info("testcluster-go: Creating cluster")
	var err error
	ctx := context.Background()

	clusterNamePrefix, err := validateClusterNamePrefix(opts.ClusterNamePrefix)
	if err != nil {
		errMsg := fmt.Sprintf("testcluster-go: Invalid cluster name prefix found: %s", err.Error())
		l.Log().Error(errMsg)
		_ = panicAtStartError(ctx, nil, err)
	}
	opts.ClusterNamePrefix = clusterNamePrefix

	cluster, err := CreateK3dCluster(ctx, opts)
	if err != nil {
		if strings.Contains(err.Error(), "port is already allocated") {
			l.Log().Error("Port is already allocated. Was another test-cluster running not properly cleaned up? The clean-up instruction might help.")
			_ = panicAtStartError(ctx, cluster, err)
			return nil
		}

		t.Errorf("Unexpected error during test setup: %s\n", err)
		_ = panicAtStartError(ctx, cluster, err)
		return nil
	}

	return cluster
}

func validateClusterNamePrefix(prefix string) (string, error) {
	generated := generatePseudoPrefix(6)
	if prefix == "" {
		return defaultPrefix + generated, nil
	}

	totalPrefix := defaultPrefix + prefix
	valid := restrictedContainerNamePattern.MatchString(totalPrefix)

	if valid {
		return totalPrefix, nil
	}

	return "", fmt.Errorf("total cluster name prefix '%s' looks invalid (valid pattern: %s)", totalPrefix, restrictedContainerNameChars)
}

func generatePseudoPrefix(length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

func registerTearDown(t *testing.T, cluster *K3dCluster) {
	if cluster == nil || cluster.clusterConfig == nil {
		t.Errorf("testcluster-go: No cluster or cluster config was found for tear down registration.")
		return
	}

	t.Cleanup(func() {
		l.Log().Debug("testcluster-go: Terminating cluster during test tear down")

		err := cluster.Terminate(context.Background())
		if err != nil {
			l.Log().Info("testcluster-go: Cluster was termination failed")
			t.Errorf("Unexpected error during test tear down: %s\n", err.Error())
			return
		}
		l.Log().Info("testcluster-go: Cluster was successfully terminated")
	})
}

func createClusterConfig(ctx context.Context, clusterName string, opts Opts) (*v1alpha5.ClusterConfig, error) {
	freeHostPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("could not find free port for port-forward: %w", err)
	}

	//TODO fix hardcoding to image reg server. It seems it may be used by different clusters but the port must be configurable.
	k3sRegistryYaml := `
my.company.registry":
  endpoint:
  - http://my.company.registry:5000
`
	const nodeFilter = "server:*"
	simpleConfig := v1alpha5.SimpleConfig{
		TypeMeta: configTypes.TypeMeta{
			Kind:       "Simple",
			APIVersion: "k3d.io/v1alpha5",
		},
		ObjectMeta: configTypes.ObjectMeta{
			Name: clusterName,
		},
		Image:   fmt.Sprintf("%s:%s", k3dTypes.DefaultK3sImageRepo, K3sVersion1_28),
		Servers: 1,
		Agents:  0,
		Options: v1alpha5.SimpleConfigOptions{
			K3dOptions: v1alpha5.SimpleConfigOptionsK3d{
				Wait:    true,
				Timeout: 60 * time.Second,
			},
			K3sOptions: v1alpha5.SimpleConfigOptionsK3s{
				ExtraArgs: []v1alpha5.K3sArgWithNodeFilters{
					{ // nodeFilters settings may correspond with the number of servers and agents above
						// TODO extract with proper naming because here goes some magic
						Arg:         "--kubelet-arg=eviction-hard=" + opts.NodeConditionEvictionHardArg,
						NodeFilters: []string{nodeFilter},
					},
				}},
		},
		// allows unpublished images-under-test to be used in the cluster
		Registries: v1alpha5.SimpleConfigRegistries{
			Create: &v1alpha5.SimpleConfigRegistryCreateConfig{
				//Name:	fmt.Sprintf("%s-%s-registry", k3dTypes.DefaultObjectNamePrefix, newCluster.Name),
				// Host:    "0.0.0.0",
				HostPort: k3dTypes.DefaultRegistryPort, // alternatively the string "random"
				// Image:    fmt.Sprintf("%s:%s", k3dTypes.DefaultRegistryImageRepo, k3dTypes.DefaultRegistryImageTag),
				Proxy: k3dTypes.RegistryProxy{
					RemoteURL: "https://registry-1.docker.io",
					Username:  "",
					Password:  "",
				},
			},
			Config: k3sRegistryYaml,
		},
		ExposeAPI: v1alpha5.SimpleExposureOpts{
			HostPort: strconv.Itoa(freeHostPort),
		},
	}

	if err := config.ProcessSimpleConfig(&simpleConfig); err != nil {
		return nil, fmt.Errorf("processing simple cluster config failed: %w", err)
	}

	clusterConfig, err := config.TransformSimpleToClusterConfig(ctx, runtimes.SelectedRuntime, simpleConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to transform cluster config: %w", err)
	}

	l.Log().Debugf("===== used cluster config =====\n%#v\n===== =====", clusterConfig)

	clusterConfig, err = config.ProcessClusterConfig(*clusterConfig)
	if err != nil {
		if err != nil {
			return nil, fmt.Errorf("processing cluster config failed: %w", err)
		}
	}

	if err = config.ValidateClusterConfig(ctx, runtimes.SelectedRuntime, *clusterConfig); err != nil {
		if err != nil {
			return nil, fmt.Errorf("failed cluster config validation: %w", err)
		}
	}

	return clusterConfig, nil
}

// CreateK3dCluster creates a completely new K8s cluster with an optional clusterNamePrefix.
func CreateK3dCluster(ctx context.Context, opts Opts) (cl *K3dCluster, err error) {
	containerRuntime := runtimes.SelectedRuntime

	clusterName := naming.MustGenerateK8sName(opts.ClusterNamePrefix)
	cl = &K3dCluster{
		containerRuntime: containerRuntime,
		ClusterName:      clusterName,
	}

	cl.clusterConfig, err = createClusterConfig(ctx, clusterName, opts)
	if err != nil {
		return nil, err
	}

	err = client.ClusterRun(ctx, containerRuntime, cl.clusterConfig)
	if err != nil {
		return nil, panicAtStartError(ctx, cl, err)
	}

	cl.kubeConfig, err = client.KubeconfigGet(ctx, containerRuntime, &cl.clusterConfig.Cluster)
	if err != nil {
		return nil, panicAtStartError(ctx, cl, err)
	}
	l.Log().Debugf("testcluster-go: ===== retrieved kube config ====\n%#v\n===== =====", cl.kubeConfig)

	err = initializeClientSet(cl)
	if err != nil {
		return nil, panicAtStartError(ctx, cl, fmt.Errorf("failed to initialize clientset: %w", err))
	}

	sa, err := createDefaultRBACForSA(ctx, cl)
	if err != nil {
		return cl, panicAtStartError(ctx, cl, fmt.Errorf("failed to create default RBAC for SA: %w", err))
	}
	cl.AdminServiceAccount = sa

	l.Log().Info("testcluster-go: Cluster was successfully created")

	err = cl.waitForDefaultSACreation(ctx)
	if err != nil {
		return cl, panicAtStartError(ctx, cl, fmt.Errorf("failed to wait for default service account: %w", err))
	}

	err = cl.checkNodeHealth(ctx, NodeHealthCheckOpts{})
	if err != nil {
		return cl, panicAtStartError(ctx, cl, fmt.Errorf("failed to check node health: %w", err))
	}

	return cl, nil
}

func initializeClientSet(c *K3dCluster) error {
	if c.kubeConfig == nil {
		panic("cluster kubeConfig went unexpectedly nil")
	}
	intermediateConfig := clientcmd.NewDefaultClientConfig(*c.kubeConfig, nil)
	clientConfig, err := intermediateConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get client config: %w", err)
	}
	c.clientConfig = clientConfig

	clientSet, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	c.clientSet = clientSet

	return nil
}

// TODO allow the user to overwrite our ClusterConfig with her own
// func CreateK3dClusterWithConfig() ...

func createDefaultRBACForSA(ctx context.Context, c *K3dCluster) (string, error) {
	const globalGalacticClusterAdminSuffix = "ford-prefect"

	clientSet, err := c.ClientSet()
	if err != nil {
		return "", err
	}

	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sa-" + globalGalacticClusterAdminSuffix,
			Namespace: DefaultNamespace,
			Labels:    map[string]string{"k3s.creator": appName},
		},
	}

	sa, err = clientSet.CoreV1().ServiceAccounts(DefaultNamespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cr-" + globalGalacticClusterAdminSuffix,
			Namespace: DefaultNamespace,
			Labels:    map[string]string{"k3s.creator": appName},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{rbacv1.VerbAll},
				APIGroups: []string{rbacv1.APIGroupAll},
				Resources: []string{rbacv1.ResourceAll},
			},
		},
	}

	clusterRole, err = clientSet.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crb-" + globalGalacticClusterAdminSuffix,
			Namespace: DefaultNamespace,
			Labels:    map[string]string{"k3s.creator": appName},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      sa.Name,
				Namespace: DefaultNamespace,
			},
		},
	}

	clusterRoleBinding, err = clientSet.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return sa.Name, nil
}

func (c *K3dCluster) waitForDefaultSACreation(ctx context.Context) error {
	errRetriable := true

	err := retry.OnError(wait.Backoff{
		Steps:    20,
		Duration: 500 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}, func(err error) bool {
		return errRetriable
	}, func() error {
		clientset, err := c.ClientSet()
		if err != nil {
			errRetriable = false
			return err
		}

		_, err = clientset.CoreV1().ServiceAccounts(DefaultNamespace).Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			l.Log().Debug("testcluster-go: no default SA found")
			return err
		}

		l.Log().Debug("testcluster-go: found default SA")
		errRetriable = true
		return nil
	})

	if err != nil {
		return fmt.Errorf("waited too long for default SA: %w", err)
	}
	return nil
}

// panicAtStartError tries to remove the cluster and panics.
// Testclusters-go is then responsible for resolving the panic.
// The cluster may be nil.
func panicAtStartError(ctx context.Context, cluster *K3dCluster, err error) error {
	if cluster == nil {
		panic(err.Error())
		return err
	}

	err2 := cluster.Terminate(ctx)
	if err2 != nil {
		l.Log().Errorf("Another error '%s' occurred while terminating the cluster due to the original error (you may want to clean-up the container landscape): %s", err2.Error(), err)
	}

	panic(err.Error())
	return err
}

// Terminate shuts down the configured k3d cluster.
func (c *K3dCluster) Terminate(ctx context.Context) error {
	err := client.ClusterDelete(ctx, c.containerRuntime, &c.clusterConfig.Cluster, k3dTypes.ClusterDeleteOpts{})
	if err != nil {
		return err
	}

	return nil
}

// ClientSet returns a K8s clientset which allows to interoperate with the cluster K8s API.
func (c *K3dCluster) ClientSet() (kubernetes.Interface, error) {
	return c.clientSet, nil
}

// NodeHealthCheckOpts customizes the way whether and how node health checks are executed.
type NodeHealthCheckOpts struct {
	// SkipCheck controls whether a node check should be executed (which is usually a good idea). Defaults to false
	SkipCheck bool
}

func (c *K3dCluster) checkNodeHealth(ctx context.Context, opts NodeHealthCheckOpts) error {
	if opts.SkipCheck {
		l.Log().Debugf("testcluster-go: skipping health-check all nodes")
		return nil
	}

	nodeInfo, err := health.FetchNodeInfo(ctx, c.clientSet)
	if err != nil {
		return err
	}

	err = health.CheckCondition(nodeInfo)
	if err != nil {
		return err
	}

	l.Log().Debugf("testcluster-go: node looks healthy for nodes %s", nodeInfo.String())
	return nil
}

// WriteKubeConfig writes a Kube Config into the directory. This directory is the
// same where other Kube Configs may reside. This is useful when a testcluster
// should be debugged manually.
func (c *K3dCluster) WriteKubeConfig(ctx context.Context, t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	kubeconf := filepath.Join(dir, "kubeconfig")
	options := &client.WriteKubeConfigOptions{
		UpdateExisting:       true,
		UpdateCurrentContext: true,
		OverwriteExisting:    true,
	}
	pathToKubeConfig, err := client.KubeconfigGetWrite(ctx, c.containerRuntime, &c.clusterConfig.Cluster, kubeconf, options)
	require.NoError(t, err)

	fmt.Println("writing temporary kubeconfig to", pathToKubeConfig)
	fmt.Printf("example: KUBECONFIG=%s kubectl get nodes\n", pathToKubeConfig)

	return pathToKubeConfig
}

func (c *K3dCluster) CtlKube(fieldManager string) (*YamlApplier, error) {
	yamlApplier, err := NewYamlApplier(c.clientConfig, fieldManager, DefaultNamespace)
	if err != nil {
		return nil, fmt.Errorf("ctlkube call failed: %w", err)
	}
	return yamlApplier, nil
}

// MustLookout creates a new Lookout that interacts with the current cluster. It
// does not return an error but panics with the found error instead.
func (c *K3dCluster) MustLookout(t *testing.T) *Lookout {
	clientSet, err := c.ClientSet()
	if err != nil {
		errMsg := fmt.Sprintf("mustLookout could not build clientSet for cluster: %s", err.Error())
		t.Errorf(errMsg)
	}

	return &Lookout{
		t: t,
		c: clientSet,
	}
}

// Lookout creates a new Lookout that interacts with the current cluster.
func (c *K3dCluster) Lookout(t *testing.T) (*Lookout, error) {
	clientSet, err := c.ClientSet()
	if err != nil {
		return nil, fmt.Errorf("lookout could not build clientSet for cluster: %w", err)
	}

	return &Lookout{
		t: t,
		c: clientSet,
	}, nil
}
