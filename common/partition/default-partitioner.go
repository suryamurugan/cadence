package partition

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgryski/go-farm"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/dynamicconfig"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/types"
	"sync"
)

type DefaultPartitionConfig struct {
	WorkflowStartZone types.ZoneName `json:"wf-start-zone"`
	RunID             string         `json:"run-id"`
}

type DefaultPartitioner struct {
	config     Config
	log        log.Logger
	drainState ZoneState
	mu         sync.RWMutex
}

type DefaultZoneStateHandler struct {
	domainCache      cache.DomainCache
	globalZoneDrains persistence.GlobalZoneDrains
	allZonesList     []types.ZoneName
	log              log.Logger
	config           Config
	mu               sync.RWMutex
}

type Config struct {
	zonalPartitioningEnabled dynamicconfig.BoolPropertyFnWithDomainFilter
}

func NewDefaultZoneStateWatcher(logger log.Logger, allZones []types.ZoneName, config Config) ZoneState {
	return &DefaultZoneStateHandler{
		log:          logger,
		allZonesList: allZones,
		config:       config,
	}
}

func NewDefaultTaskResolver(logger log.Logger) Partitioner {
	return &DefaultPartitioner{
		log: logger,
	}
}

func (r *DefaultPartitioner) IsDrained(ctx context.Context, domain string, zone types.ZoneName) (bool, error) {
	state, err := r.drainState.Get(ctx, domain, zone)
	if err != nil {
		return false, fmt.Errorf("could not determine if drained: %w", err)
	}
	return state.Status == types.ZoneDrainStatusDrained, nil
}

func (r *DefaultPartitioner) IsDrainedByDomainID(ctx context.Context, domainID string, zone types.ZoneName) (bool, error) {
	state, err := r.drainState.GetByDomainID(ctx, domainID, zone)
	if err != nil {
		return false, fmt.Errorf("could not determine if drained: %w", err)
	}
	return state.Status == types.ZoneDrainStatusDrained, nil
}

func (r *DefaultPartitioner) GetTaskZone(ctx context.Context, DomainID string, key types.PartitionConfig) (*types.ZoneName, error) {
	partitionData := DefaultPartitionConfig{}
	err := json.Unmarshal(key, &partitionData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode partition config: %w", err)
	}

	isDrained, err := r.IsDrained(ctx, DomainID, partitionData.WorkflowStartZone)
	if err != nil {
		return nil, fmt.Errorf("failed to determine if a zone is drained: %w", err)
	}

	if isDrained {
		zones, err := r.drainState.ListAll(ctx, DomainID)
		if err != nil {
			return nil, fmt.Errorf("failed to list all zones: %w", err)
		}
		zone := pickZoneAfterDrain(zones, partitionData)
		return &zone, nil
	}

	return &partitionData.WorkflowStartZone, nil
}

func (z *DefaultZoneStateHandler) ListAll(ctx context.Context, domainID string) ([]types.ZonePartition, error) {
	var out []types.ZonePartition

	for _, zone := range z.allZonesList {
		zoneData, err := z.Get(ctx, domainID, zone)
		if err != nil {
			return nil, fmt.Errorf("failed to get zone during listing: %w", err)
		}
		out = append(out, *zoneData)
	}

	return out, nil
}

func (z *DefaultZoneStateHandler) GetByDomainID(ctx context.Context, domainID string, zone types.ZoneName) (*types.ZonePartition, error) {
	domain, err := z.domainCache.GetDomainByID(domainID)
	if err != nil {
		return nil, fmt.Errorf("could not resolve domain in zone handler: %w", err)
	}
	return z.Get(ctx, domain.GetInfo().Name, zone)
}

// Get the statue of a zone, with respect to both domain and global drains. Domain-specific drains override global config
func (z *DefaultZoneStateHandler) Get(ctx context.Context, domain string, zone types.ZoneName) (*types.ZonePartition, error) {
	if !z.config.zonalPartitioningEnabled(domain) {
		return &types.ZonePartition{
			Name:   zone,
			Status: types.ZoneDrainStatusHealthy,
		}, nil
	}

	domainData, err := z.domainCache.GetDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("could not resolve domain in zone handler: %w", err)
	}
	cfg, ok := domainData.GetInfo().ZoneConfig[zone]
	if ok && cfg.Status == types.ZoneDrainStatusDrained {
		return &cfg, nil
	}

	drains, err := z.globalZoneDrains.GetClusterDrains(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not resolve global drains in zone handler: %w", err)
	}
	globalCfg, ok := drains[zone]
	if ok {
		return &globalCfg, nil
	}

	return &types.ZonePartition{
		Name:   zone,
		Status: types.ZoneDrainStatusHealthy,
	}, nil
}

// Simple deterministic zone picker
// which will pick a random healthy zone and place the workflow there
func pickZoneAfterDrain(zones []types.ZonePartition, wfConfig DefaultPartitionConfig) types.ZoneName {
	var availableZones []types.ZoneName
	for _, zone := range zones {
		if zone.Status == types.ZoneDrainStatusHealthy {
			availableZones = append(availableZones, zone.Name)
		}
	}
	hashv := farm.Hash32([]byte(wfConfig.RunID))
	return availableZones[int(hashv)%len(availableZones)]
}
