// Copyright (c) Facebook, Inc. and its affiliates.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// +build integration_storage

package dblocker

import (
	"os"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"

	"github.com/facebookincubator/contest/pkg/target"
	"github.com/facebookincubator/contest/pkg/types"
	"github.com/facebookincubator/contest/pkg/xcontext/bundles/logrusctx"
	"github.com/facebookincubator/contest/pkg/xcontext/logger"
	"github.com/facebookincubator/contest/plugins/targetlocker/dblocker"
	"github.com/facebookincubator/contest/tests/integ/common"
)

var (
	testBatchSize = 3

	ctx = logrusctx.NewContext(logger.LevelDebug)

	jobID                                 = types.JobID(123)
	defaultJobTargetManagerAcquireTimeout = 2 * time.Second

	allTargets = []*target.Target{
		&target.Target{ID: "001"},
		&target.Target{ID: "002"},
		&target.Target{ID: "003"},
		&target.Target{ID: "004"},
	}
	oneTarget  = []*target.Target{allTargets[0]}
	twoTargets = []*target.Target{allTargets[0], allTargets[1]}

	tl *dblocker.DBLocker

	tlClock = clock.NewMock()
)

func TestMain(m *testing.M) {
	// tests reset the database, which makes the locker yell all the time,
	// disable for the integration tests

	var err error
	tl, err = dblocker.New(
		common.GetDatabaseURI(),
		dblocker.WithClock(tlClock),
		dblocker.WithMaxBatchSize(testBatchSize),
	)
	if err != nil {
		panic(err)
	}
	// mysql doesn't like epoch, so jump forward a bit
	tlClock.Add(1 * time.Hour)
	os.Exit(m.Run())
}

func TestNew(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NotNil(t, tl)
	assert.IsType(t, &dblocker.DBLocker{}, tl)
}

func TestLockInvalidJobIDAndNoTargets(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.Error(t, tl.Lock(ctx, 0, defaultJobTargetManagerAcquireTimeout, nil))
}

func TestLockValidJobIDAndNoTargets(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, nil))
}

func TestLockValidJobIDAndNoTargets2(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, []*target.Target{}))
}

func TestLockInvalidJobIDAndOneTarget(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.Error(t, tl.Lock(ctx, 0, defaultJobTargetManagerAcquireTimeout, oneTarget))
}

func TestLockValidJobIDAndEmptyIDTarget(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.Error(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, []*target.Target{&target.Target{ID: ""}}))
}

func TestLockValidJobIDAndOneTarget(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, oneTarget))
}

func TestLockValidJobIDAndTwoTargets(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets))
}

func TestLockReentrantLock(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, allTargets))
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, oneTarget))
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, allTargets))
}

func TestLockReentrantLockDifferentJobID(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, allTargets))
	assert.Error(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, oneTarget))
	assert.Error(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, twoTargets))
	assert.Error(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, allTargets))
	assert.Error(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, []*target.Target{allTargets[3]}))
}

func TestUnlockInvalidJobIDAndNoTargets(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Unlock(ctx, jobID, nil))
}

func TestUnlockValidJobIDAndNoTargets(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Unlock(ctx, jobID, nil))
}

func TestUnlockInvalidJobIDAndOneTarget(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.Error(t, tl.Unlock(ctx, 0, oneTarget))
}

func TestUnlockValidJobIDAndOneTarget(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Unlock(ctx, jobID, oneTarget))
}

func TestUnlockValidJobIDAndTwoTargets(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Unlock(ctx, jobID, twoTargets))
}

func TestLockUnlockSameJobID(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, allTargets))
	assert.NoError(t, tl.Unlock(ctx, jobID, allTargets))
}

func TestLockUnlockDifferentJobID(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, allTargets))
	// this does not error, but will also not release the lock...
	assert.NoError(t, tl.Unlock(ctx, jobID+1, allTargets))
	// ... so it cannot be acquired by job+1
	assert.Error(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, twoTargets))
}

func TestTryLockOne(t *testing.T) {
	tl.ResetAllLocks(ctx)
	res, err := tl.TryLock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, oneTarget, 1)
	assert.NoError(t, err)
	assert.Equal(t, oneTarget[0].ID, res[0])
}

func TestTryLockTwo(t *testing.T) {
	tl.ResetAllLocks(ctx)
	res, err := tl.TryLock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets, 2)
	assert.NoError(t, err)
	// order is not guaranteed
	assert.Contains(t, res, twoTargets[0].ID)
	assert.Contains(t, res, twoTargets[1].ID)
}

func TestTryLockSome(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets))
	res, err := tl.TryLock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, allTargets, uint(len(allTargets)))
	assert.NoError(t, err)
	// asked for all, got some
	assert.Equal(t, 2, len(res))
	assert.Contains(t, res, allTargets[2].ID)
	assert.Contains(t, res, allTargets[3].ID)
}

func TestTryLockSameJob(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets))
	// job is the same, so we get all 4
	res, err := tl.TryLock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, allTargets, uint(len(allTargets)))
	assert.NoError(t, err)
	assert.Equal(t, 4, len(res))
	assert.Contains(t, res, allTargets[0].ID)
	assert.Contains(t, res, allTargets[1].ID)
	assert.Contains(t, res, allTargets[2].ID)
	assert.Contains(t, res, allTargets[3].ID)
}

func TestInMemoryTryLockZeroLimited(t *testing.T) {
	tl.ResetAllLocks(ctx)
	// only request one
	res, err := tl.TryLock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets, 0)
	assert.NoError(t, err)
	// it is allowed to set the limit to zero
	assert.Equal(t, len(res), 0)
}

func TestTryLockTwoHigherLimit(t *testing.T) {
	tl.ResetAllLocks(ctx)
	// limit is just an upper bound, can be higher
	res, err := tl.TryLock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets, 100)
	assert.NoError(t, err)
	// order is not guaranteed
	assert.Contains(t, res, twoTargets[0].ID)
	assert.Contains(t, res, twoTargets[1].ID)
}

func TestInMemoryTryLockOneLimited(t *testing.T) {
	tl.ResetAllLocks(ctx)
	// only request one
	res, err := tl.TryLock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets, 1)
	assert.NoError(t, err)
	assert.Equal(t, len(res), 1)
	// API doesn't require it, but this locker guarantees order
	// so the first one should have been locked,
	// the second not because limit was 1
	assert.Contains(t, res, twoTargets[0].ID)
	assert.NotContains(t, res, twoTargets[1].ID)
}

func TestInMemoryTryLockOneOfTwo(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, oneTarget))
	// now tryLock both with other ID
	res, err := tl.TryLock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, twoTargets, 2)
	assert.NoError(t, err)
	// should have locked 1 but not 0
	assert.NotContains(t, res, twoTargets[0].ID)
	assert.Contains(t, res, twoTargets[1].ID)
}

func TestInMemoryTryLockNoneOfTwo(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets))
	// now tryLock both with other ID
	res, err := tl.TryLock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, twoTargets, 2)
	// should have locked zero targets, but no error
	assert.NoError(t, err)
	assert.Empty(t, res)
}

func TestRefreshLocks(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.RefreshLocks(ctx, jobID, twoTargets))
}

func TestRefreshLocksTwice(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.RefreshLocks(ctx, jobID, twoTargets))
	assert.NoError(t, tl.RefreshLocks(ctx, jobID, twoTargets))
}

func TestRefreshLocksOneThenTwo(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.RefreshLocks(ctx, jobID, oneTarget))
	assert.NoError(t, tl.RefreshLocks(ctx, jobID, twoTargets))
}

func TestRefreshLocksTwoThenOne(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.RefreshLocks(ctx, jobID, twoTargets))
	assert.NoError(t, tl.RefreshLocks(ctx, jobID, oneTarget))
}

func TestLockExpiry(t *testing.T) {
	tl.ResetAllLocks(ctx)
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets))
	// getting them immediately fails for other owner
	assert.Error(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, twoTargets))
	tlClock.Add(3 * time.Second)
	// expired, now it should work
	assert.NoError(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, twoTargets))
}

func TestRefreshMultiple(t *testing.T) {
	// not super happy with this test, it is timing sensitive
	tl.ResetAllLocks(ctx)
	// now for the actual test
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, twoTargets))
	tlClock.Add(1500 * time.Millisecond)
	// they are not expired yet, extend both
	assert.NoError(t, tl.RefreshLocks(ctx, jobID, twoTargets))
	tlClock.Add(1 * time.Second)
	// if they were refreshed properly, they are still valid and attempts to get them must fail
	assert.Error(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, []*target.Target{allTargets[0]}))
	assert.Error(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, []*target.Target{allTargets[1]}))
}

func TestLockingTransactional(t *testing.T) {
	tl.ResetAllLocks(ctx)
	// lock the second target
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, []*target.Target{allTargets[1]}))
	// try to lock both with another owner (this fails as expected)
	assert.Error(t, tl.Lock(ctx, jobID+1, defaultJobTargetManagerAcquireTimeout, twoTargets))
	// API says target one should remain unlocked because Lock() is transactional
	// this means it can be locked by the first owner
	assert.NoError(t, tl.Lock(ctx, jobID, defaultJobTargetManagerAcquireTimeout, []*target.Target{allTargets[0]}))
}
