package gedis

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/jbenet/go-base58"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/geoip"
)

func TestRegisterFarm(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	args := Args{
		"farm": directory.TfgridFarm1{
			ThreebotID:      "farm-id",
			Name:            "my-node",
			Email:           "azmy@test.com",
			WalletAddresses: []string{"addr1", "addr2"},
		},
	}

	conn.On("Do", "default.farms.register", mustMarshal(t, args)).
		Return(mustMarshal(t, Args{"farm_id": 123}), nil)

	id, err := gedis.RegisterFarm(
		pkg.StrIdentifier("farm-id"),
		"my-node",
		"azmy@test.com",
		[]string{"addr1", "addr2"},
	)

	require.NoError(err)
	require.Equal(id, "123")
	conn.AssertCalled(t, "Close")
}

func TestRegisterNode(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	l, err := geoip.Fetch()
	require.NoError(err)

	args := Args{
		"node": directory.TfgridNode2{
			NodeID:       "node-1",
			FarmID:       "farm-1",
			OsVersion:    "v1.1.0",
			PublicKeyHex: hex.EncodeToString(base58.Decode("node-1")),
			Location: directory.TfgridLocation1{
				Longitude: l.Longitute,
				Latitude:  l.Latitude,
				Continent: l.Continent,
				Country:   l.Country,
				City:      l.City,
			},
		},
	}

	conn.On("Do", "default.nodes.add", mustMarshal(t, args)).
		Return(mustMarshal(t, Args{"node_id": "node-1"}), nil)

	id, err := gedis.RegisterNode(
		pkg.StrIdentifier("node-1"),
		pkg.StrIdentifier("farm-1"),
		"v1.1.0",
		l,
	)

	require.NoError(err)
	require.Equal(id, "node-1")
	conn.AssertCalled(t, "Close")
}

func TestListNode(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	args := Args{
		"farm_id": "farm-1",
		"country": "eg",
		"city":    "cairo",
	}

	conn.On("Do", "default.nodes.list", mustMarshal(t, args)).
		Return(mustMarshal(t, Args{"nodes": []directory.TfgridNode2{
			{NodeID: "node-1"},
			{NodeID: "node-2"},
		}}), nil)

	nodes, err := gedis.ListNode(
		pkg.StrIdentifier("farm-1"), "eg", "cairo",
	)

	require.NoError(err)
	require.Len(nodes, 2)
	require.Equal(nodes[0].NodeID, "node-1")
	conn.AssertCalled(t, "Close")
}

func TestGetNode(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	args := Args{
		"node_id": "node-1",
	}

	conn.On("Do", "default.nodes.get", mustMarshal(t, args)).
		Return(mustMarshal(t, directory.TfgridNode2{
			NodeID: "node-1",
		}), nil)

	node, err := gedis.GetNode(
		pkg.StrIdentifier("node-1"),
	)

	require.NoError(err)
	require.Equal(node.NodeID, "node-1")
	conn.AssertCalled(t, "Close")
}

func TestGetFarm(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	args := Args{
		"farm_id": 100,
	}

	conn.On("Do", "default.farms.get", mustMarshal(t, args)).
		Return(mustMarshal(t, directory.TfgridFarm1{
			ID:   "100",
			Name: "farm-1",
		}), nil)

	farm, err := gedis.GetFarm(
		pkg.StrIdentifier("100"),
	)

	require.NoError(err)
	require.Equal("100", farm.ID)
	require.Equal("farm-1", farm.Name)
	conn.AssertCalled(t, "Close")
}

func TestListFarm(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	args := Args{
		"country": "eg",
		"city":    "cairo",
	}

	conn.On("Do", "default.farms.list", mustMarshal(t, args)).
		Return(mustMarshal(t, Args{"farms": []directory.TfgridFarm1{
			{ID: "1", Name: "farm-1"},
			{ID: "2", Name: "farm-2"},
		}}), nil)

	nodes, err := gedis.ListFarm("eg", "cairo")

	require.NoError(err)
	require.Len(nodes, 2)
	require.Equal(nodes[0].ID, "1")
	require.Equal(nodes[0].Name, "farm-1")
	conn.AssertCalled(t, "Close")
}

func TestUpdateGenericNodeCapacity(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	node := pkg.StrIdentifier("node-1")

	args := Args{
		"node_id": node.Identity(),
		"resource": directory.TfgridNodeResourceAmount1{
			Mru: 1,
			Cru: 2,
			Hru: 3,
			Sru: 4,
		},
	}

	const captype = "total"
	action := fmt.Sprintf("default.nodes.update_%s_capacity", captype)
	conn.On("Do", action, mustMarshal(t, args)).
		Return(nil, nil)

	err := gedis.updateGenericNodeCapacity(
		"total",
		node,
		1, 2, 3, 4)

	require.NoError(err)
	conn.AssertCalled(t, "Close")
}