package privateapi

import (
	"context"
	"testing"

	"github.com/ledgerwatch/erigon-lib/gointerfaces"
	"github.com/ledgerwatch/erigon-lib/gointerfaces/remote"
	types2 "github.com/ledgerwatch/erigon-lib/gointerfaces/types"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
	"github.com/stretchr/testify/require"
)

// Hashes
var (
	startingHeadHash = common.HexToHash("0x1")
	payload1Hash     = common.HexToHash("a1726a8541a53f098b21f58e9d1c63a450b8acbecdce0510162f8257be790320")
	payload2Hash     = common.HexToHash("a19013b8b5f95ffaa008942fd2f04f72a3ae91f54eb64a3f6f8c6630db742aef")
	payload3Hash     = common.HexToHash("8aa80e6d2270ae7273e87fa5752d94b94a0f05da60f99fd4b8e36ca767cbae91")
)

// Payloads
var (
	mockPayload1 *types2.ExecutionPayload = &types2.ExecutionPayload{
		ParentHash:    gointerfaces.ConvertHashToH256(common.HexToHash("0x2")),
		BlockHash:     gointerfaces.ConvertHashToH256(payload1Hash),
		ReceiptRoot:   gointerfaces.ConvertHashToH256(common.HexToHash("0x3")),
		StateRoot:     gointerfaces.ConvertHashToH256(common.HexToHash("0x4")),
		Random:        gointerfaces.ConvertHashToH256(common.HexToHash("0x0b3")),
		LogsBloom:     gointerfaces.ConvertBytesToH2048(make([]byte, 256)),
		ExtraData:     gointerfaces.ConvertHashToH256(common.Hash{}),
		BaseFeePerGas: gointerfaces.ConvertHashToH256(common.HexToHash("0x0b3")),
		BlockNumber:   100,
		GasLimit:      52,
		GasUsed:       4,
		Timestamp:     4,
		Coinbase:      gointerfaces.ConvertAddressToH160(common.HexToAddress("0x1")),
		Transactions:  make([][]byte, 0),
	}
	mockPayload2 *types2.ExecutionPayload = &types2.ExecutionPayload{
		ParentHash:    gointerfaces.ConvertHashToH256(payload1Hash),
		BlockHash:     gointerfaces.ConvertHashToH256(payload2Hash),
		ReceiptRoot:   gointerfaces.ConvertHashToH256(common.HexToHash("0x3")),
		StateRoot:     gointerfaces.ConvertHashToH256(common.HexToHash("0x4")),
		Random:        gointerfaces.ConvertHashToH256(common.HexToHash("0x0b3")),
		LogsBloom:     gointerfaces.ConvertBytesToH2048(make([]byte, 256)),
		ExtraData:     gointerfaces.ConvertHashToH256(common.Hash{}),
		BaseFeePerGas: gointerfaces.ConvertHashToH256(common.HexToHash("0x0b3")),
		BlockNumber:   101,
		GasLimit:      52,
		GasUsed:       4,
		Timestamp:     4,
		Coinbase:      gointerfaces.ConvertAddressToH160(common.HexToAddress("0x1")),
		Transactions:  make([][]byte, 0),
	}
	mockPayload3 = &types2.ExecutionPayload{
		ParentHash:    gointerfaces.ConvertHashToH256(startingHeadHash),
		BlockHash:     gointerfaces.ConvertHashToH256(payload3Hash),
		ReceiptRoot:   gointerfaces.ConvertHashToH256(common.HexToHash("0x3")),
		StateRoot:     gointerfaces.ConvertHashToH256(common.HexToHash("0x4")),
		Random:        gointerfaces.ConvertHashToH256(common.HexToHash("0x0b3")),
		LogsBloom:     gointerfaces.ConvertBytesToH2048(make([]byte, 256)),
		ExtraData:     gointerfaces.ConvertHashToH256(common.Hash{}),
		BaseFeePerGas: gointerfaces.ConvertHashToH256(common.HexToHash("0x0b3")),
		BlockNumber:   51,
		GasLimit:      52,
		GasUsed:       4,
		Timestamp:     4,
		Coinbase:      gointerfaces.ConvertAddressToH160(common.HexToAddress("0x1")),
		Transactions:  make([][]byte, 0),
	}
)

func makeTestDb(ctx context.Context, db kv.RwDB) {
	tx, _ := db.BeginRw(ctx)
	rawdb.WriteHeadBlockHash(tx, startingHeadHash)
	rawdb.WriteHeaderNumber(tx, startingHeadHash, 50)
	rawdb.MarkTransition(tx, 0)
	_ = tx.Commit()
}

func TestMockDownloadRequest(t *testing.T) {
	db := memdb.New()
	ctx := context.Background()
	require := require.New(t)

	makeTestDb(ctx, db)
	reverseDownloadCh := make(chan types.Block)
	statusCh := make(chan ExecutionStatus)
	waitingForHeaders := true

	backend := NewEthBackendServer(ctx, nil, db, nil, nil, &params.ChainConfig{TerminalTotalDifficulty: common.Big1}, reverseDownloadCh, statusCh, &waitingForHeaders)

	var err error
	var reply *remote.EngineExecutePayloadReply
	done := make(chan bool)

	go func() {
		reply, err = backend.EngineExecutePayloadV1(ctx, mockPayload1)
		done <- true
	}()

	<-reverseDownloadCh
	statusCh <- ExecutionStatus{
		HeadHash: startingHeadHash,
		Status:   Syncing,
	}
	waitingForHeaders = false
	<-done
	require.NoError(err)
	require.Equal(reply.Status, string(Syncing))
	replyHash := gointerfaces.ConvertH256ToHash(reply.LatestValidHash)
	require.Equal(replyHash[:], startingHeadHash[:])

	// If we get another request we dont need to process it with processDownloadCh and ignore it and return Syncing status
	go func() {
		reply, err = backend.EngineExecutePayloadV1(ctx, mockPayload2)
		done <- true
	}()

	<-done
	// Same result as before
	require.NoError(err)
	require.Equal(reply.Status, string(Syncing))
	replyHash = gointerfaces.ConvertH256ToHash(reply.LatestValidHash)
	require.Equal(replyHash[:], startingHeadHash[:])

	// However if we simulate that we finish reverse downloading the chain by updating the head, we just execute 1:1
	tx, _ := db.BeginRw(ctx)
	rawdb.WriteHeadBlockHash(tx, payload1Hash)
	rawdb.WriteHeaderNumber(tx, payload1Hash, 100)
	_ = tx.Commit()
	// Now we try to sync the next payload again
	go func() {
		reply, err = backend.EngineExecutePayloadV1(ctx, mockPayload2)
		done <- true
	}()

	<-done

	require.NoError(err)
	require.Equal(reply.Status, string(Syncing))
	replyHash = gointerfaces.ConvertH256ToHash(reply.LatestValidHash)
	require.Equal(replyHash[:], startingHeadHash[:])
}

func TestMockValidExecution(t *testing.T) {
	db := memdb.New()
	ctx := context.Background()
	require := require.New(t)

	makeTestDb(ctx, db)

	reverseDownloadCh := make(chan types.Block)
	statusCh := make(chan ExecutionStatus)
	waitingForHeaders := true

	backend := NewEthBackendServer(ctx, nil, db, nil, nil, &params.ChainConfig{TerminalTotalDifficulty: common.Big1}, reverseDownloadCh, statusCh, &waitingForHeaders)

	var err error
	var reply *remote.EngineExecutePayloadReply
	done := make(chan bool)

	go func() {
		reply, err = backend.EngineExecutePayloadV1(ctx, mockPayload3)
		done <- true
	}()

	<-reverseDownloadCh

	statusCh <- ExecutionStatus{
		HeadHash: payload3Hash,
		Status:   Valid,
	}
	<-done

	require.NoError(err)
	require.Equal(reply.Status, string(Valid))
	replyHash := gointerfaces.ConvertH256ToHash(reply.LatestValidHash)
	require.Equal(replyHash[:], payload3Hash[:])
}

func TestMockInvalidExecution(t *testing.T) {
	db := memdb.New()
	ctx := context.Background()
	require := require.New(t)

	makeTestDb(ctx, db)

	reverseDownloadCh := make(chan types.Block)
	statusCh := make(chan ExecutionStatus)

	waitingForHeaders := true
	backend := NewEthBackendServer(ctx, nil, db, nil, nil, &params.ChainConfig{TerminalTotalDifficulty: common.Big1}, reverseDownloadCh, statusCh, &waitingForHeaders)

	var err error
	var reply *remote.EngineExecutePayloadReply
	done := make(chan bool)

	go func() {
		reply, err = backend.EngineExecutePayloadV1(ctx, mockPayload3)
		done <- true
	}()

	<-reverseDownloadCh
	// Simulate invalid status
	statusCh <- ExecutionStatus{
		HeadHash: startingHeadHash,
		Status:   Invalid,
	}
	<-done

	require.NoError(err)
	require.Equal(reply.Status, string(Invalid))
	replyHash := gointerfaces.ConvertH256ToHash(reply.LatestValidHash)
	require.Equal(replyHash[:], startingHeadHash[:])
}

func TestInvalidRequest(t *testing.T) {
	db := memdb.New()
	ctx := context.Background()
	require := require.New(t)

	makeTestDb(ctx, db)

	reverseDownloadCh := make(chan types.Block)
	statusCh := make(chan ExecutionStatus)
	waitingForHeaders := true

	backend := NewEthBackendServer(ctx, nil, db, nil, nil, &params.ChainConfig{TerminalTotalDifficulty: common.Big1}, reverseDownloadCh, statusCh, &waitingForHeaders)

	var err error

	done := make(chan bool)

	go func() {
		// The payload is malformed, some fields are not set
		_, err = backend.EngineExecutePayloadV1(ctx, &types2.ExecutionPayload{
			BaseFeePerGas: gointerfaces.ConvertHashToH256(common.HexToHash("0x0b3")),
			BlockNumber:   51,
			GasLimit:      52,
			GasUsed:       4,
			Timestamp:     4,
			Coinbase:      gointerfaces.ConvertAddressToH160(common.HexToAddress("0x1")),
			Transactions:  make([][]byte, 0),
		})
		done <- true
	}()

	<-done

	require.Equal(err.Error(), "invalid execution payload")
}

func TestNoTTD(t *testing.T) {
	db := memdb.New()
	ctx := context.Background()
	require := require.New(t)

	makeTestDb(ctx, db)

	reverseDownloadCh := make(chan types.Block)
	statusCh := make(chan ExecutionStatus)
	waitingForHeaders := true

	backend := NewEthBackendServer(ctx, nil, db, nil, nil, &params.ChainConfig{}, reverseDownloadCh, statusCh, &waitingForHeaders)

	var err error

	done := make(chan bool)

	go func() {
		_, err = backend.EngineExecutePayloadV1(ctx, &types2.ExecutionPayload{
			ParentHash:    gointerfaces.ConvertHashToH256(common.HexToHash("0x2")),
			BlockHash:     gointerfaces.ConvertHashToH256(common.HexToHash("0x3")),
			ReceiptRoot:   gointerfaces.ConvertHashToH256(common.HexToHash("0x4")),
			StateRoot:     gointerfaces.ConvertHashToH256(common.HexToHash("0x4")),
			Random:        gointerfaces.ConvertHashToH256(common.HexToHash("0x0b3")),
			LogsBloom:     gointerfaces.ConvertBytesToH2048(make([]byte, 256)),
			ExtraData:     gointerfaces.ConvertHashToH256(common.Hash{}),
			BaseFeePerGas: gointerfaces.ConvertHashToH256(common.HexToHash("0x0b3")),
			BlockNumber:   51,
			GasLimit:      52,
			GasUsed:       4,
			Timestamp:     4,
			Coinbase:      gointerfaces.ConvertAddressToH160(common.HexToAddress("0x1")),
			Transactions:  make([][]byte, 0),
		})
		done <- true
	}()

	<-done

	require.Equal(err.Error(), "not a proof-of-stake chain")
}
