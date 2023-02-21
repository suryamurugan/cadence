package persistence

import (
	"context"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/types"
	"time"
)

type GlobalZoneDrains interface {
	GetClusterDrains(ctx context.Context) (map[types.ZoneName]types.ZonePartition, error)
	SetClusterDrains(ctx context.Context, partition types.ZonePartition) error
}

type globalZoneDrainsImpl struct {
	serializer  PayloadSerializer
	persistence ConfigStore
	logger      log.Logger
}

func (z *globalZoneDrainsImpl) SetClusterDrains(ctx context.Context, partition types.ZonePartition) error {
	panic("not implemented")
	z.persistence.UpdateConfig(ctx, &InternalConfigStoreEntry{
		RowType:   ZonalConfig,
		Version:   0,
		Timestamp: time.Time{},
		Values:    nil,
	})
	return nil
}

func (z *globalZoneDrainsImpl) GetClusterDrains(ctx context.Context) (map[types.ZoneName]types.ZonePartition, error) {
	panic("not implemented")
	z.persistence.FetchConfig(ctx, ZonalConfig)
	return nil, nil
}
