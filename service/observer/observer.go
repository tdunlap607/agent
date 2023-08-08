// Package observer implements the Observer service for Flow.
package observer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/grafana/agent/component"
	"github.com/grafana/agent/pkg/river"
	"github.com/grafana/agent/pkg/river/encoding/riverjson"
	"github.com/grafana/agent/service"
)

// ServiceName defines the name used for the Observer service.
const ServiceName = "observer"

type Arguments struct {
	RefreshFrequency time.Duration `river:"refresh_frequency,attr,optional"`
	RemoteEndpoint   string        `river:"remote_endpoint,attr"`
}

var _ river.Defaulter = (*Arguments)(nil)

// DefaultArguments holds default values for Arguments.
var DefaultArguments = Arguments{
	RefreshFrequency: time.Minute,
}

// SetToDefault implements river.Defaulter.
func (args *Arguments) SetToDefault() {
	*args = DefaultArguments
}

type Observer struct {
	mtx          sync.Mutex
	args         Arguments
	configUpdate chan struct{}
}

var _ service.Service = (*Observer)(nil)

// New returns a new, unstarted instance of the HTTP service.
func New(args Arguments) *Observer {
	return &Observer{
		mtx:          sync.Mutex{},
		args:         args,
		configUpdate: make(chan struct{}, 1),
	}
}

// Data implements service.Service.
func (*Observer) Data() any {
	return nil
}

// Definition implements service.Service.
func (*Observer) Definition() service.Definition {
	return service.Definition{
		Name:       ServiceName,
		ConfigType: Arguments{},
		DependsOn:  nil, // observer has no dependencies.
	}
}

// Run implements service.Service.
func (o *Observer) Run(ctx context.Context, host service.Host) error {
	for {
		o.mtx.Lock()
		refreshFrequency := o.args.RefreshFrequency
		o.mtx.Unlock()

		o.observe(host)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(refreshFrequency):
			o.observe(host)
		case <-o.configUpdate:
			continue
		}
	}
}

func (o *Observer) observe(host service.Host) {
	components, err := host.ListComponents("", component.InfoOptions{
		GetHealth:    true,
		GetArguments: true,
		GetExports:   true,
		GetDebugInfo: true,
	})
	if err != nil {
		//TODO: Log a warning and continue?
	}

	getAgentState(components)

	//TODO: Acquire the config mutex where necessary

}

func getAgentState(components []*component.Info) []Component {
	res := []Component{}

	parentId := uint(0)
	for _, cInfo := range components {
		// ComponentDetail should start at index 0 for each component
		idCounter := parentId + 1

		componentDetail := []riverjson.ComponentDetail{}

		// Add the arguments
		componentDetail = append(componentDetail, getTopLevelComponentDetail("arguments", parentId, &idCounter))
		componentDetail = append(componentDetail, riverjson.GetComponentDetail(cInfo.Arguments, parentId, &idCounter)...)

		// Add the exports
		componentDetail = append(componentDetail, getTopLevelComponentDetail("exports", parentId, &idCounter))
		componentDetail = append(componentDetail, riverjson.GetComponentDetail(cInfo.Exports, parentId, &idCounter)...)

		// Add the debug info
		componentDetail = append(componentDetail, getTopLevelComponentDetail("debug_info", parentId, &idCounter))
		componentDetail = append(componentDetail, riverjson.GetComponentDetail(cInfo.DebugInfo, parentId, &idCounter)...)

		componentState := Component{
			ID:       cInfo.ID.LocalID,
			ModuleID: cInfo.ID.ModuleID,
			Health: Health{
				Health:     cInfo.Health.Health.String(),
				Message:    cInfo.Health.Message,
				UpdateTime: cInfo.Health.UpdateTime,
			},
			ComponentDetail: componentDetail,
		}

		res = append(res, componentState)
	}

	return res
}

func getTopLevelComponentDetail(componentName string, parentId uint, idCounter *uint) riverjson.ComponentDetail {
	res := riverjson.ComponentDetail{
		ID:         *idCounter,
		ParentID:   parentId,
		Name:       componentName,
		Label:      "",
		RiverType:  "",
		RiverValue: []byte{},
	}
	*idCounter += 1
	return res
}

// Update implements service.Service.
func (o *Observer) Update(newConfig any) error {
	cfg, ok := newConfig.(Arguments)
	if !ok {
		return fmt.Errorf("invalid configuration passed to the %q service", ServiceName)
	}

	o.mtx.Lock()
	defer o.mtx.Unlock()

	o.args = cfg

	// Only send an update signal if there isn't one already, so that Update() doesn't block
	//TODO: Not sure how thread safe this is, but it should do for now.
	if len(o.configUpdate) == 0 {
		o.configUpdate <- struct{}{}
	}

	return nil
}

//-----------------------------------------
// TODO: Remove these structs later
//-----------------------------------------

// Metadata
type AgentState struct {
	ID     string
	Labels map[string]string
}

// RG type
type Component struct {
	ID              string                      `parquet:"id"`
	ModuleID        string                      `parquet:"module_id"`
	Health          Health                      `parquet:"health"`
	ComponentDetail []riverjson.ComponentDetail `parquet:"component_detail"`
}

type Health struct {
	Health     string    `parquet:"state"`
	Message    string    `parquet:"message"`
	UpdateTime time.Time `parquet:"update_time"`
}