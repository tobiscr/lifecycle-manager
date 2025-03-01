package catalog

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/lifecycle-manager/operator/api/v1alpha1"
	"github.com/kyma-project/lifecycle-manager/operator/pkg/remote"
)

type Sync struct {
	Catalog
}

func NewSync(client client.Client, settings Settings) *Sync {
	return &Sync{Catalog: New(client, settings)}
}

func (s *Sync) Cleanup(
	ctx context.Context,
) error {
	return s.Catalog.Delete(ctx)
}

func (s *Sync) Run(
	ctx context.Context,
	kyma *v1alpha1.Kyma,
	moduleTemplateList *v1alpha1.ModuleTemplateList,
) error {
	if kyma.Spec.Sync.Enabled {
		if err := s.syncRemote(ctx, kyma, moduleTemplateList); err != nil {
			return err
		}
	} else {
		if err := s.syncLocal(ctx, kyma, moduleTemplateList); err != nil {
			return err
		}
	}
	return nil
}

func (s *Sync) syncRemote(
	ctx context.Context,
	controlPlaneKyma *v1alpha1.Kyma,
	moduleTemplateList *v1alpha1.ModuleTemplateList,
) error {
	syncContext, err := remote.InitializeKymaSynchronizationContext(ctx, s.Catalog.Client(), controlPlaneKyma)
	if err != nil {
		return fmt.Errorf("could not initialize remote context before updating remote kyma: %w", err)
	}

	return New(syncContext.RuntimeClient, s.Catalog.Settings()).CreateOrUpdate(ctx, moduleTemplateList.Items)
}

func (s *Sync) syncLocal(
	ctx context.Context,
	_ *v1alpha1.Kyma,
	moduleTemplateList *v1alpha1.ModuleTemplateList,
) error {
	return s.Catalog.CreateOrUpdate(ctx, moduleTemplateList.Items)
}
