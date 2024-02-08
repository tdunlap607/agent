package labelstore

import (
	"math"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/value"
	"github.com/stretchr/testify/require"
)

func TestAddingMarker(t *testing.T) {
	mapping := New(log.NewNopLogger(), prometheus.DefaultRegisterer)
	l := labels.Labels{}
	l = append(l, labels.Label{
		Name:  "__name__",
		Value: "test",
	})
	globalID := mapping.getOrAddGlobalRefID(l)
	shouldBeSameGlobalID := mapping.getOrAddGlobalRefID(l)
	require.True(t, globalID == shouldBeSameGlobalID)
	require.Len(t, mapping.labelsHashToGlobal, 1)
}

func TestAddingDifferentMarkers(t *testing.T) {
	mapping := New(log.NewNopLogger(), prometheus.DefaultRegisterer)
	l := labels.Labels{}
	l = append(l, labels.Label{
		Name:  "__name__",
		Value: "test",
	})
	l2 := labels.Labels{}
	l2 = append(l2, labels.Label{
		Name:  "__name__",
		Value: "roar",
	})
	globalID := mapping.getOrAddGlobalRefID(l)
	shouldBeDifferentID := mapping.getOrAddGlobalRefID(l2)
	require.True(t, globalID != shouldBeDifferentID)
	require.Len(t, mapping.labelsHashToGlobal, 2)
}

func TestAddingLocalMapping(t *testing.T) {
	mapping := New(log.NewNopLogger(), prometheus.DefaultRegisterer)
	l := labels.Labels{}
	l = append(l, labels.Label{
		Name:  "__name__",
		Value: "test",
	})

	globalID := mapping.getOrAddGlobalRefID(l)
	shouldBeSameGlobalID := mapping.GetOrAddLink("1", 1, l)
	require.True(t, globalID == shouldBeSameGlobalID)
	require.Len(t, mapping.labelsHashToGlobal, 1)
	require.Len(t, mapping.mappings, 1)
	require.True(t, mapping.mappings["1"].RemoteWriteID == "1")
	require.True(t, mapping.mappings["1"].globalToLocal[shouldBeSameGlobalID] == 1)
	require.True(t, mapping.mappings["1"].localToGlobal[1] == shouldBeSameGlobalID)
}

func TestAddingLocalMappings(t *testing.T) {
	mapping := New(log.NewNopLogger(), prometheus.DefaultRegisterer)
	l := labels.Labels{}
	l = append(l, labels.Label{
		Name:  "__name__",
		Value: "test",
	})

	globalID := mapping.getOrAddGlobalRefID(l)
	shouldBeSameGlobalID := mapping.GetOrAddLink("1", 1, l)
	shouldBeSameGlobalID2 := mapping.GetOrAddLink("2", 1, l)
	require.True(t, globalID == shouldBeSameGlobalID)
	require.True(t, globalID == shouldBeSameGlobalID2)
	require.Len(t, mapping.labelsHashToGlobal, 1)
	require.Len(t, mapping.mappings, 2)

	require.True(t, mapping.mappings["1"].RemoteWriteID == "1")
	require.True(t, mapping.mappings["1"].globalToLocal[shouldBeSameGlobalID] == 1)
	require.True(t, mapping.mappings["1"].localToGlobal[1] == shouldBeSameGlobalID)

	require.True(t, mapping.mappings["2"].RemoteWriteID == "2")
	require.True(t, mapping.mappings["2"].globalToLocal[shouldBeSameGlobalID2] == 1)
	require.True(t, mapping.mappings["2"].localToGlobal[1] == shouldBeSameGlobalID2)
}

func TestAddingLocalMappingsWithoutCreatingGlobalUpfront(t *testing.T) {
	mapping := New(log.NewNopLogger(), prometheus.DefaultRegisterer)
	l := labels.Labels{}
	l = append(l, labels.Label{
		Name:  "__name__",
		Value: "test",
	})

	shouldBeSameGlobalID := mapping.GetOrAddLink("1", 1, l)
	shouldBeSameGlobalID2 := mapping.GetOrAddLink("2", 1, l)
	require.True(t, shouldBeSameGlobalID2 == shouldBeSameGlobalID)
	require.Len(t, mapping.labelsHashToGlobal, 1)
	require.Len(t, mapping.mappings, 2)

	require.True(t, mapping.mappings["1"].RemoteWriteID == "1")
	require.True(t, mapping.mappings["1"].globalToLocal[shouldBeSameGlobalID] == 1)
	require.True(t, mapping.mappings["1"].localToGlobal[1] == shouldBeSameGlobalID)

	require.True(t, mapping.mappings["2"].RemoteWriteID == "2")
	require.True(t, mapping.mappings["2"].globalToLocal[shouldBeSameGlobalID2] == 1)
	require.True(t, mapping.mappings["2"].localToGlobal[1] == shouldBeSameGlobalID2)
}

func TestStaleness(t *testing.T) {
	mapping := New(log.NewNopLogger(), prometheus.DefaultRegisterer)
	l := labels.Labels{}
	l = append(l, labels.Label{
		Name:  "__name__",
		Value: "test",
	})
	l2 := labels.Labels{}
	l2 = append(l2, labels.Label{
		Name:  "__name__",
		Value: "test2",
	})

	global1 := mapping.ConvertToSeries(0, math.Float64frombits(value.StaleNaN), l)
	_ = mapping.GetOrAddLink("2", 1, l2)
	mapping.HandleStaleMarkers([]*Series{global1})
	require.Len(t, mapping.staleGlobals, 1)
	require.Len(t, mapping.labelsHashToGlobal, 2)
	staleDuration = 1 * time.Millisecond
	time.Sleep(10 * time.Millisecond)
	mapping.checkAndRemoveStaleMarkers()
	require.Len(t, mapping.staleGlobals, 0)
	require.Len(t, mapping.labelsHashToGlobal, 1)
}
