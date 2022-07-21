package manifests

import (
	"fmt"
	"os"
	"path/filepath"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/agent"
)

var (
	clusterDeploymentFilename = filepath.Join(clusterManifestDir, "cluster-deployment.yaml")
)

// ClusterDeployment generates the cluster-deployment.yaml file.
type ClusterDeployment struct {
	File   *asset.File
	Config *hivev1.ClusterDeployment
}

var _ asset.WritableAsset = (*ClusterDeployment)(nil)

// Name returns a human friendly name for the asset.
func (*ClusterDeployment) Name() string {
	return "ClusterDeployment Config"
}

// Dependencies returns all of the dependencies directly needed to generate
// the asset.
func (*ClusterDeployment) Dependencies() []asset.Asset {
	return []asset.Asset{
		&agent.OptionalInstallConfig{},
	}
}

// Generate generates the ClusterDeployment manifest.
func (cd *ClusterDeployment) Generate(dependencies asset.Parents) error {
	installConfig := &agent.OptionalInstallConfig{}
	dependencies.Get(installConfig)

	if installConfig.Config != nil {
		clusterDeployment := &hivev1.ClusterDeployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterDeployment",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      getClusterDeploymentName(installConfig),
				Namespace: getObjectMetaNamespace(installConfig),
			},
			Spec: hivev1.ClusterDeploymentSpec{
				ClusterName: getClusterDeploymentName(installConfig),
				BaseDomain:  installConfig.Config.BaseDomain,
				PullSecretRef: &corev1.LocalObjectReference{
					Name: getPullSecretName(installConfig),
				},
				ClusterInstallRef: &hivev1.ClusterInstallLocalReference{
					Group:   "extensions.hive.openshift.io",
					Version: "v1beta1",
					Kind:    "AgentClusterInstall",
					Name:    getAgentClusterInstallName(installConfig),
				},
			},
		}

		cd.Config = clusterDeployment
		clusterDeploymentData, err := yaml.Marshal(clusterDeployment)
		if err != nil {
			return errors.Wrap(err, "failed to marshal agent installer ClusterDeployment")
		}

		cd.File = &asset.File{
			Filename: clusterDeploymentFilename,
			Data:     clusterDeploymentData,
		}

	}

	return cd.finish()
}

// Files returns the files generated by the asset.
func (cd *ClusterDeployment) Files() []*asset.File {
	if cd.File != nil {
		return []*asset.File{cd.File}
	}
	return []*asset.File{}
}

// Load returns ClusterDeployment asset from the disk.
func (cd *ClusterDeployment) Load(f asset.FileFetcher) (bool, error) {

	file, err := f.FetchByName(clusterDeploymentFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrap(err, fmt.Sprintf("failed to load %s file", clusterDeploymentFilename))
	}

	config := &hivev1.ClusterDeployment{}
	if err := yaml.UnmarshalStrict(file.Data, config); err != nil {
		return false, errors.Wrapf(err, "failed to unmarshal %s", clusterDeploymentFilename)
	}

	cd.File, cd.Config = file, config
	if err = cd.finish(); err != nil {
		return false, err
	}

	return true, nil
}

func (cd *ClusterDeployment) finish() error {

	if cd.Config == nil {
		return errors.New("missing configuration or manifest file")
	}

	return nil
}
