package keccak

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"sync"
	"time"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Challenger interface {
	Challenge(ctx context.Context, block rpcblock.Block, oracle Oracle, preimages []keccakTypes.LargePreimageMetaData) error
}

type OracleSource interface {
	Oracles() []keccakTypes.LargePreimageOracle
}

type Metrics interface {
	RecordLargePreimageCount(count int)
}

type LargePreimageScheduler struct {
	log        log.Logger
	m          Metrics
	cl         faultTypes.ClockReader
	ch         chan rpcblock.Block
	oracles    OracleSource
	challenger Challenger
	cancel     func()
	wg         sync.WaitGroup
}

func NewLargePreimageScheduler(
	logger log.Logger,
	m Metrics,
	cl faultTypes.ClockReader,
	oracleSource OracleSource,
	challenger Challenger) *LargePreimageScheduler {
	return &LargePreimageScheduler{
		log:        logger,
		m:          m,
		cl:         cl,
		ch:         make(chan rpcblock.Block, 1),
		oracles:    oracleSource,
		challenger: challenger,
	}
}

func (s *LargePreimageScheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go s.run(ctx)
}

func (s *LargePreimageScheduler) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *LargePreimageScheduler) run(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case block := <-s.ch:
			if err := s.verifyPreimages(ctx, block); err != nil {
				s.log.Error("Failed to verify large preimages", "block", block, "err", err)
			}
		}
	}
}

// Schedule queues a preimage check pinned to the given L1 block. The block is referenced by
// number rather than hash so the check works on L1s whose block hash is not keccak(rlp(header)).
func (s *LargePreimageScheduler) Schedule(_ common.Hash, blockNumber uint64) error {
	select {
	case s.ch <- rpcblock.ByNumber(blockNumber):
	default:
		s.log.Trace("Skipping preimage check while already processing")
	}
	return nil
}

func (s *LargePreimageScheduler) verifyPreimages(ctx context.Context, block rpcblock.Block) error {
	var err error
	for _, oracle := range s.oracles.Oracles() {
		err = errors.Join(err, s.verifyOraclePreimages(ctx, oracle, block))
	}
	return err
}

func (s *LargePreimageScheduler) verifyOraclePreimages(ctx context.Context, oracle keccakTypes.LargePreimageOracle, block rpcblock.Block) error {
	preimages, err := oracle.GetActivePreimages(ctx, block)
	if err != nil {
		return err
	}
	s.m.RecordLargePreimageCount(len(preimages))
	period, err := oracle.ChallengePeriod(ctx)
	if err != nil {
		return fmt.Errorf("failed to load challenge period: %w", err)
	}
	toVerify := make([]keccakTypes.LargePreimageMetaData, 0, len(preimages))
	for _, preimage := range preimages {
		if preimage.ShouldVerify(s.cl.Now(), time.Duration(period)*time.Second) {
			toVerify = append(toVerify, preimage)
		}
	}
	return s.challenger.Challenge(ctx, block, oracle, toVerify)
}
