package types

const (
	ZoneDrainStatusInvalid ZoneStatus = iota
	ZoneDrainStatusHealthy
	ZoneDrainStatusDrained
)

// A ZoneName is a subdivision of a 'region', such as a subset of racks in a datacentre or a division of
// traffic which there is a need for logical separation for resilience, but these subdivisions still operate within
// the databases' ability to operate consistently.
type ZoneName string
type ZoneStatus int

// PartitionConfig is a key/value based set of configuration for partitioning traffic. Intended to be a key/value pair
// of data encoded in JSON or whatever encoding suits. This is intentionally opaque and to be passed blindly
// to the partitioner of choice as it may contain business-specific types.
//
// Example of the intent:
// partitionCfg := []byte(`{"wf-start-zone": "zone123", "userid: "1234", "weighting": 0.5}`)
// which, for example, may allow the partitioner to choose to split traffic based on where the workflow started, or
// the user, or any arbitrary other configuration
type PartitionConfig []byte

type ZonePartition struct {
	Name   ZoneName
	Status ZoneStatus
}
