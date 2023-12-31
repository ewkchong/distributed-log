package discovery_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/serf/serf"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
	. "proglog/internal/discovery"
)

func TestMembership(t *testing.T) {
	m, handler := setupMember(t, nil)
	m, _ = setupMember(t, m)
	m, _ = setupMember(t, m)

	//  NOTE: we require that in 3 seconds, checking every 250ms, we have had 2 join events, 3 members, and no leaves
	require.Eventually(t, func() bool {
		return 2 == len(handler.joins) &&
				3 == len(m[0].Members()) &&
				0 == len(handler.leaves)
	}, 3*time.Second, 250*time.Millisecond)

	//  NOTE: set the third member to leave the cluster
	require.NoError(t, m[2].Leave())

	//  NOTE: check that the second member has left
	require.Eventually(t, func() bool{
		return 2 == len(handler.joins) &&
				3 == len(m[0].Members()) &&
				serf.StatusLeft == m[0].Members()[2].Status &&
				1 == len(handler.leaves)
	}, 3*time.Second, 250*time.Millisecond)

	require.Equal(t, fmt.Sprintf("%d", 2), <-handler.leaves)
}

func setupMember(t *testing.T, members []*Membership) (
	[]*Membership, *handler,
) {
	id := len(members)

	// NOTE: get 1 free port
	ports := dynaport.Get(1)
	addr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
	tags := map[string]string{
		"rpc_addr": addr,
	}
	c := Config{
		NodeName: fmt.Sprintf("%d", id),
		BindAddr: addr,
		Tags: tags,
	}
	h := &handler{}

	if len(members) == 0 {
		h.joins = make(chan map[string]string, 3)
		h.leaves = make(chan string, 3)
	} else {
		c.StartJoinAddrs = []string{
			members[0].BindAddr,
		}
	}
	m, err := New(h, c)
	require.NoError(t, err)
	members = append(members, m)
	return members, h
}

type handler struct {
	joins chan map[string]string
	leaves chan string
}

//  NOTE: h must implement Join and Leave, interface defined in membership.go
func (h *handler) Join(id, addr string) error {
	if h.joins != nil {
		h.joins <- map[string]string{
			"id": id,
			"addr": addr,
		}
	}
	return nil
}

func (h *handler) Leave(id string) error {
	if h.leaves != nil {
		h.leaves <- id
	}
	return nil
}
