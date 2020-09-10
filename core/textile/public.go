package textile

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	api_buckets_pb "github.com/textileio/textile/api/buckets/pb"

	"github.com/FleekHQ/space-daemon/core/textile/bucket"

	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/core/thread"
	bc "github.com/textileio/textile/api/buckets/client"
)

// Get a public bucket on hub. Public bucket has no encryption and its content should be accessible directly via ipfs/ipns
// Only use this bucket for items that is okay to be publicly shared
func (tc *textileClient) GetPublicShareBucket(ctx context.Context) (Bucket, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.getOrCreatePublicBucket(ctx, defaultPublicShareBucketSlug)
}

func (tc *textileClient) getOrCreatePublicBucket(ctx context.Context, bucketSlug string) (Bucket, error) {
	ctx, dbId, err := tc.getPublicShareBucketContext(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	// find if bucket exists
	buckets, err := tc.hb.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get public bucket")
	}

	if buckets != nil {
		for _, bucketRoot := range buckets.Roots {
			if bucketRoot.Name == bucketSlug {
				return bucket.New(
					bucketRoot,
					tc.getPublicShareBucketContext,
					tc.hb,
				), nil
			}
		}
	}

	// else create bucketRoot
	bucketRoot, err := tc.createPublicBucket(ctx, *dbId, bucketSlug)
	if err != nil {
		return nil, err
	}

	newB := bucket.New(
		bucketRoot,
		tc.getPublicShareBucketContext,
		tc.hb,
	)

	return newB, nil
}

func (tc *textileClient) getPublicShareBucketContext(ctx context.Context, bucketSlug string) (context.Context, *thread.ID, error) {
	dbId, err := tc.getPublicShareThread(ctx)
	if err != nil {
		return nil, nil, err
	}
	ctx, err = utils.GetThreadContext(ctx, bucketSlug, *dbId, true, tc.kc, tc.hubAuth)
	if err != nil {
		return nil, nil, err
	}

	return ctx, dbId, nil
}

// Creates a public bucket for current user.
func (tc *textileClient) createPublicBucket(ctx context.Context, dbId thread.ID, bucketSlug string) (*api_buckets_pb.Root, error) {
	log.Debug("Creating a new public bucket")

	hubCtx, _, err := tc.getBucketContext(ctx, utils.CastDbIDToString(dbId), bucketSlug, true, nil)
	if err != nil {
		return nil, err
	}

	b, err := tc.hb.Create(hubCtx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	return b.Root, nil
}

// Creates a remote hub thread for the public sharing bucket
func (tc *textileClient) getPublicShareThread(ctx context.Context) (*thread.ID, error) {
	log.Debug("createPublicShareThread: Generating a new threadID ...")
	var err error
	ctx, err = tc.getHubCtx(ctx)
	if err != nil {
		return nil, err
	}

	//dbId, err := tc.getPublicShareThreadID()
	//if err != nil {
	//	return nil, err
	//}

	dbId := thread.NewIDV1(thread.Raw, 32)

	// check if db exists
	_, err = tc.ht.GetDBInfo(ctx, dbId)
	if err != nil && !strings.Contains(err.Error(), "thread not found") {
		return nil, errors.Wrap(err, "failed to fetch db info")
	}

	// create new db
	if err := tc.ht.NewDB(ctx, dbId); err != nil {
		return nil, err
	}
	log.Debug("Public share thread created")
	return &dbId, nil
}

// returns a deterministic threadId for the current logged in user
func (tc *textileClient) getPublicShareThreadID() (thread.ID, error) {
	return utils.NewDeterministicThreadID(tc.kc, utils.PublicthreadThreadVariant)
}
