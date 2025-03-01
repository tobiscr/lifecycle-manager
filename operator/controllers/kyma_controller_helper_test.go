package controllers_test

import (
	"encoding/json"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/lifecycle-manager/operator/api/v1alpha1"
	sampleCRDv1alpha1 "github.com/kyma-project/lifecycle-manager/operator/config/samples/component-integration-installed/crd/v1alpha1" //nolint:lll
	"github.com/kyma-project/lifecycle-manager/operator/controllers"
	"github.com/kyma-project/lifecycle-manager/operator/pkg/parsed"
	"github.com/kyma-project/lifecycle-manager/operator/pkg/watch"
	manifestV1alpha1 "github.com/kyma-project/manifest-operator/operator/api/v1alpha1"
)

func NewTestKyma(name string) *v1alpha1.Kyma {
	return &v1alpha1.Kyma{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       string(v1alpha1.KymaKind),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.KymaSpec{
			Modules: []v1alpha1.Module{},
			Channel: v1alpha1.DefaultChannel,
		},
	}
}

func RegisterDefaultLifecycleForKyma(kyma *v1alpha1.Kyma) {
	BeforeEach(func() {
		Expect(controlPlaneClient.Create(ctx, kyma)).Should(Succeed())
	})
	AfterEach(func() {
		Expect(controlPlaneClient.Delete(ctx, kyma)).Should(Succeed())
	})
}

func IsKymaInState(kyma *v1alpha1.Kyma, state v1alpha1.State) func() bool {
	return func() bool {
		kymaFromCluster, err := GetKyma(controlPlaneClient, kyma)
		if err != nil || kymaFromCluster.Status.State != state {
			return false
		}
		return true
	}
}

func GetKymaState(kyma *v1alpha1.Kyma) func() string {
	return func() string {
		createdKyma, err := GetKyma(controlPlaneClient, kyma)
		if err != nil {
			return ""
		}
		return string(createdKyma.Status.State)
	}
}

func GetKymaConditions(kyma *v1alpha1.Kyma) func() []metav1.Condition {
	return func() []metav1.Condition {
		createdKyma, err := GetKyma(controlPlaneClient, kyma)
		if err != nil {
			return []metav1.Condition{}
		}
		return createdKyma.Status.Conditions
	}
}

func UpdateModuleState(
	kyma *v1alpha1.Kyma, moduleTemplate *v1alpha1.ModuleTemplate, state v1alpha1.State,
) func() error {
	return func() error {
		component, err := getModule(kyma, moduleTemplate)
		Expect(err).ShouldNot(HaveOccurred())
		component.Object[watch.Status] = map[string]any{watch.State: string(state)}
		return k8sManager.GetClient().Status().Update(ctx, component)
	}
}

func ModuleExists(kyma *v1alpha1.Kyma, moduleTemplate *v1alpha1.ModuleTemplate) func() bool {
	return func() bool {
		_, err := getModule(kyma, moduleTemplate)
		return err == nil
	}
}

func ModuleNotExist(kyma *v1alpha1.Kyma, moduleTemplate *v1alpha1.ModuleTemplate) func() bool {
	return func() bool {
		_, err := getModule(kyma, moduleTemplate)
		return k8serrors.IsNotFound(err)
	}
}

func SKRModuleExistWithOverwrites(kyma *v1alpha1.Kyma, moduleTemplate *v1alpha1.ModuleTemplate) func() string {
	return func() string {
		module, err := getModule(kyma, moduleTemplate)
		Expect(err).ToNot(HaveOccurred())
		body, err := json.Marshal(module.Object["spec"])
		Expect(err).ToNot(HaveOccurred())
		manifestSpec := manifestV1alpha1.ManifestSpec{}
		err = json.Unmarshal(body, &manifestSpec)
		Expect(err).ToNot(HaveOccurred())
		body, err = json.Marshal(manifestSpec.Resource.Object["spec"])
		Expect(err).ToNot(HaveOccurred())
		skrModuleSpec := sampleCRDv1alpha1.SKRModuleSpec{}
		err = json.Unmarshal(body, &skrModuleSpec)
		Expect(err).ToNot(HaveOccurred())
		return skrModuleSpec.InitKey
	}
}

func getModule(
	kyma *v1alpha1.Kyma,
	moduleTemplate *v1alpha1.ModuleTemplate,
) (*unstructured.Unstructured, error) {
	component := moduleTemplate.Spec.Data.DeepCopy()
	if moduleTemplate.Spec.Target == v1alpha1.TargetRemote {
		component.SetKind("Manifest")
	}
	err := controlPlaneClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      parsed.CreateModuleName(moduleTemplate.GetLabels()[v1alpha1.ModuleName], kyma.GetName()),
	}, component)
	if err != nil {
		return nil, err
	}
	return component, nil
}

func GetKyma(
	testClient client.Client,
	kyma *v1alpha1.Kyma,
) (*v1alpha1.Kyma, error) {
	kymaInCluster := &v1alpha1.Kyma{}
	err := testClient.Get(ctx, client.ObjectKeyFromObject(kyma), kymaInCluster)
	if err != nil {
		return nil, err
	}
	return kymaInCluster, nil
}

func RemoteKymaExists(remoteClient client.Client, kyma *v1alpha1.Kyma) func() error {
	return func() error {
		_, err := GetKyma(remoteClient, kyma)
		return err
	}
}

func getCatalog(
	clnt client.Client,
	kyma *v1alpha1.Kyma,
) (*v1.ConfigMap, error) {
	catalog := &v1.ConfigMap{}
	catalog.SetName(controllers.CatalogName)
	catalog.SetNamespace(kyma.GetNamespace())
	err := clnt.Get(ctx, client.ObjectKeyFromObject(catalog), catalog)
	if err != nil {
		return nil, err
	}
	return catalog, nil
}

func CatalogExists(clnt client.Client, kyma *v1alpha1.Kyma) func() error {
	return func() error {
		_, err := getCatalog(clnt, kyma)
		return err
	}
}
