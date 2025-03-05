// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package autopilot

import (
	"fmt"
	"sort"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/raft"
)

// reconcile calculates and then applies promotions and demotions
func (a *Autopilot) reconcile() error {
	if !a.ReconciliationEnabled() {
		return nil
	}

	conf := a.delegate.AutopilotConfig()
	if conf == nil {
		return nil
	}

	// grab the current state while locked
	state := a.GetState()

	if state == nil || state.Leader == "" {
		return fmt.Errorf("cannot reconcile Raft server voting rights without a valid autopilot state")
	}

	// have the promoter calculate the required Raft changeset.
	changes := a.promoter.CalculatePromotionsAndDemotions(conf, state)

	// Apply the demotion to a failed server first, but only a single one per a
	// reconciliation round and if the number of voters is odd. In that case we
	// don't want to start with the promotions, since that temporarily inflates
	// quorum, which could lead to cluster failure if more voters fail.
	if len(state.Voters)%2 != 0 {
		if _, err := a.applyDemotions(state, changes, true); err != nil {
			return err
		}
		// We are proceeding with promotions regardless of the outcome of the demotion,
		// as long as there were no errors. This is to avoid having to wait for another
		// reconciliation round before the promotions take place.
	}

	// apply the promotions, if we did apply any then stop here
	// as we do not want to apply the demotions at the same time
	// as a means of preventing cluster instability.
	if done, err := a.applyPromotions(state, changes); done {
		return err
	}

	// apply the demotions, if we did apply any then stop here
	// as we do not want to transition leadership and do demotions
	// at the same time. This is a preventative measure to maintain
	// cluster stability.
	if done, err := a.applyDemotions(state, changes, false); done {
		return err
	}

	// if no leadership transfer is desired then we can exit the method now.
	if changes.Leader == "" || changes.Leader == state.Leader {
		return nil
	}

	// lookup the server we want to transfer leadership to
	srv, ok := state.Servers[changes.Leader]
	if !ok {
		return fmt.Errorf("cannot transfer leadership to an unknown server with ID %s", changes.Leader)
	}

	// perform the leadership transfer
	return a.leadershipTransfer(changes.Leader, srv.Server.Address)
}

// applyPromotions will apply all the promotions in the RaftChanges parameter.
//
// IDs in the change set will be ignored if:
// * The server isn't tracked in the provided state
// * The server already has voting rights
// * The server is not healthy
//
// If any servers were promoted this function returns true for the bool value.
func (a *Autopilot) applyPromotions(state *State, changes RaftChanges) (bool, error) {
	promoted := false
	for _, change := range changes.Promotions {
		srv, found := state.Servers[change]
		if !found {
			a.logger.Debug("Ignoring promotion of server as it is not in the autopilot state", "id", change)
			// this shouldn't be able to happen but is a nice safety measure against the
			// delegate doing something less than desirable
			continue
		}

		if srv.HasVotingRights() {
			// There is no need to promote as this server is already a voter.
			// No logging is needed here as this could be a very common case
			// where the promoter just returns a lists of server ids that should
			// be voters and non-voters without caring about which ones currently
			// already are in that state.
			a.logger.Debug("Not promoting server that already has voting rights", "id", change)
			continue
		}

		if !srv.Health.Healthy {
			// do not promote unhealthy servers
			a.logger.Debug("Ignoring promotion of unhealthy server", "id", change)
			continue
		}

		a.logger.Info("Promoting server", "id", srv.Server.ID, "address", srv.Server.Address, "name", srv.Server.Name)

		if err := a.addVoter(srv.Server.ID, srv.Server.Address); err != nil {
			return true, fmt.Errorf("failed promoting server %s: %v", srv.Server.ID, err)
		}

		promoted = true
	}

	// when we promoted anything we return true to indicate that the promotion/demotion applying
	// process is finished to prevent promotions and demotions in the same round. This is what
	// autopilot within Consul used to do, so I am keeping the behavior the same for now.
	return promoted, nil
}

// applyDemotions will apply the demotions in the RaftChanges parameter either to all or
// just unhealthy servers, based on the value of the demoteSingleFailedServer parameter:
//   - If demoteSingleFailedServer is true, then a single demotion will be applied
//     to an unhealthy server and the function will return. We limit this to a
//     single demotion to prevent violating the minimum quorum setting.
//   - If demoteSingleFailedServer is false, then all of the demotions will be
//     applied regardless of the health status of the servers.
//
// IDs in the change set will be ignored if:
//   - The server isn't tracked in the provided state.
//   - The server does not have voting rights.
//   - The server is healthy and the demoteSingleFailedServer
//     parameter is true, i.e. when we are demoting a failed server, healthy servers
//     will be ignored.
//
// If any servers were demoted this function returns true for the bool value.
func (a *Autopilot) applyDemotions(state *State, changes RaftChanges, demoteSingleFailedServer bool) (bool, error) {
	demoted := false
	for _, change := range changes.Demotions {
		srv, found := state.Servers[change]
		if !found {
			a.logger.Debug("Ignoring demotion of server as it is not in the autopilot state", "id", change)
			// this shouldn't be able to happen but is a nice safety measure against the
			// delegate doing something less than desirable
			continue
		}

		if srv.State == RaftNonVoter {
			// There is no need to demote as this server is already a non-voter.
			// No logging is needed here as this could be a very common case
			// where the promoter just returns a lists of server ids that should
			// be voters and non-voters without caring about which ones currently
			// already are in that state.
			a.logger.Debug("Ignoring demotion of server that is already a non-voter", "id", change)
			continue
		}

		if demoteSingleFailedServer && srv.Health.Healthy {
			a.logger.Debug("Ignoring demotion of healthy server during failed server demotion process", "id", change)
			continue
		}

		a.logger.Info("Demoting server", "id", srv.Server.ID, "address", srv.Server.Address, "name", srv.Server.Name)

		if err := a.demoteVoter(srv.Server.ID); err != nil {
			return true, fmt.Errorf("failed demoting server %s: %v", srv.Server.ID, err)
		}

		demoted = true

		// We only want to demote one failed server at a time to prevent violating the minimum quorum setting.
		if demoteSingleFailedServer {
			return demoted, nil
		}
	}

	// similarly to applyPromotions here we want to stop the process and prevent leadership
	// transfer when any demotions took place. Basically we want to ensure the cluster is
	// stable before doing the transfer
	return demoted, nil
}

// getFailedServers aggregates all the information about servers that the consuming application believes are in
// a failed/left state (indicated by the NodeStatus field on the Server type) as well as stale servers that are
// in the raft configuration but not know to the consuming application. This function will do nothing with
// that information and is purely to collect the data.
func (a *Autopilot) getFailedServers() (*FailedServers, *voterRegistry, error) {
	staleRaftServers := make(map[raft.ServerID]raft.Server)
	raftConfig, err := a.getRaftConfiguration()
	if err != nil {
		return nil, nil, err
	}

	// Populate a map of all the raft servers. We will
	// remove some later on from the map leaving us with
	// just the stale servers.
	registry := newVoterRegistry()

	for _, server := range raftConfig.Servers {
		staleRaftServers[server.ID] = server
		registry.eligibility[server.ID] = &voterEligibility{
			currentVoter: server.Suffrage == raft.Voter,
		}
	}

	var failed FailedServers

	for id, srv := range a.delegate.KnownServers() {
		raftSrv, found := staleRaftServers[id]
		if found {
			delete(staleRaftServers, id)
		} else {
			// This server was known to the application,
			// but not in the Raft config, so will be ignored
			continue
		}

		// Update the potential suffrage using the supplied predicate.
		v := registry.eligibility[id]
		v.setPotentialVoter(a.promoter.IsPotentialVoter(srv.NodeType))

		if srv.NodeStatus != NodeAlive {
			if found && raftSrv.Suffrage == raft.Voter {
				failed.FailedVoters = append(failed.FailedVoters, srv)
			} else if found {
				failed.FailedNonVoters = append(failed.FailedNonVoters, srv)
			}
		}
	}

	for id, srv := range staleRaftServers {
		if srv.Suffrage == raft.Voter {
			failed.StaleVoters = append(failed.StaleVoters, id)
		} else {
			failed.StaleNonVoters = append(failed.StaleNonVoters, id)
		}
	}

	sort.Slice(failed.StaleNonVoters, func(i, j int) bool {
		return failed.StaleNonVoters[i] < failed.StaleNonVoters[j]
	})
	sort.Slice(failed.StaleVoters, func(i, j int) bool {
		return failed.StaleVoters[i] < failed.StaleVoters[j]
	})
	sort.Slice(failed.FailedNonVoters, func(i, j int) bool {
		return failed.FailedNonVoters[i].ID < failed.FailedNonVoters[j].ID
	})
	sort.Slice(failed.FailedVoters, func(i, j int) bool {
		return failed.FailedVoters[i].ID < failed.FailedVoters[j].ID
	})

	return &failed, registry, nil
}

// pruneDeadServers will find stale raft servers and failed servers as indicated by the consuming application
// and remove them. For stale raft servers this means removing them from the Raft configuration. For failed
// servers this means issuing RemoveFailedNode calls to the delegate. All stale/failed non-voters will be
// removed first. Then stale voters and finally failed servers. For servers with voting rights we will
// cap the number removed so that we do not remove too many at a time and do not remove nodes to the
// point where the number of voters would be below the MinQuorum value from the autopilot config.
// Additionally, the delegate will be consulted to determine if all the removals should be done and
// can filter the failed servers listings if need be.
func (a *Autopilot) pruneDeadServers() error {
	if !a.ReconciliationEnabled() {
		return nil
	}

	conf := a.delegate.AutopilotConfig()
	if conf == nil || !conf.CleanupDeadServers {
		return nil
	}

	state := a.GetState()

	failed, vr, err := a.getFailedServers()
	if err != nil || failed == nil {
		return err
	}

	failed = a.promoter.FilterFailedServerRemovals(conf, state, failed)

	// Remove servers in order of increasing precedence (and update the registry)
	// Rules:
	// 1. Deal with non-voters first as their removal shouldn't impact cluster stability.
	// 2. Handle 'stale' before 'failed' in order to make progress towards the applications desired server set.

	// remove stale non-voters
	toRemove := a.adjudicateRemoval(failed.StaleNonVoters, vr)
	if err = a.removeStaleServers(toRemove); err != nil {
		return err
	}
	vr.remove(toRemove...)

	// Remove stale voters
	toRemove = a.adjudicateRemoval(failed.StaleVoters, vr)
	if err = a.removeStaleServers(toRemove); err != nil {
		return err
	}
	vr.remove(toRemove...)

	// remove failed non-voters
	failedNonVoters := vr.filter(failed.FailedNonVoters)
	toRemove = a.adjudicateRemoval(failedNonVoters, vr)
	a.removeFailedServers(failed.getFailed(toRemove, false))
	vr.remove(toRemove...)

	// remove failed voters
	failedVoters := vr.filter(failed.FailedVoters)
	toRemove = a.adjudicateRemoval(failedVoters, vr)
	a.removeFailedServers(failed.getFailed(toRemove, true))
	vr.remove(toRemove...)

	return nil
}

func (a *Autopilot) adjudicateRemoval(ids []raft.ServerID, vr *voterRegistry) []raft.ServerID {
	var result []raft.ServerID
	initialPotentialVoters := vr.potentialVoters()
	removedPotentialVoters := 0
	maxRemoval := (initialPotentialVoters - 1) / 2
	minQuorum := a.delegate.AutopilotConfig().MinQuorum

	for _, id := range ids {
		v := vr.eligibility[id]

		if v != nil && v.isPotentialVoter() && initialPotentialVoters-removedPotentialVoters-1 < int(minQuorum) {
			a.logger.Debug("will not remove server node as it would leave less voters than the minimum number allowed", "id", id, "min", minQuorum)
		} else if v.isCurrentVoter() && maxRemoval < 1 {
			a.logger.Debug("will not remove server node as removal of a majority of voting servers is not safe", "id", id)
		} else if v != nil && v.isPotentialVoter() {
			maxRemoval--
			// We need to track how many voters we have removed from the registry
			// to ensure the total remaining potential voters is accurate
			removedPotentialVoters++
			result = append(result, id)
		} else {
			result = append(result, id)
		}
	}

	return result
}

func (a *Autopilot) removeStaleServer(id raft.ServerID) error {
	a.logger.Debug("removing server by ID", "id", id)
	future := a.raft.RemoveServer(id, 0, 0)
	if err := future.Error(); err != nil {
		a.logger.Error("failed to remove raft server", "id", id, "error", err)
		return err
	}
	a.logger.Info("removed server", "id", id)
	return nil
}

func (a *Autopilot) removeStaleServers(toRemove []raft.ServerID) error {
	var result error

	for _, id := range toRemove {
		err := a.removeStaleServer(id)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}

func (a *Autopilot) removeFailedServers(toRemove []*Server) {
	for _, srv := range toRemove {
		a.delegate.RemoveFailedServer(srv)
	}
}
