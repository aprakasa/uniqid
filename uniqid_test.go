package uniqid

import (
	"crypto/rand"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

// TestNewGenerator tests the New function
func TestNewGenerator(t *testing.T) {
	// Test with nil config
	gen, err := New(nil)
	if err != nil {
		t.Fatalf("New(nil) failed: %v", err)
	}
	if gen == nil {
		t.Fatal("New(nil) returned nil generator")
	}

	// Test with a specific shard ID
	gen, err = New(&Config{ShardID: 123})
	if err != nil {
		t.Fatalf("New({ShardID: 123}) failed: %v", err)
	}
	if gen.shard != 123 {
		t.Errorf("Expected shardID 123, got %d", gen.shard)
	}

	// Test with an invalid shard ID
	_, err = New(&Config{ShardID: 2000})
	if err == nil {
		t.Error("Expected error for invalid shardID, got nil")
	}

	// Test with auto shard ID
	gen, err = New(&Config{ShardID: -1})
	if err != nil {
		t.Fatalf("New({ShardID: -1}) failed: %v", err)
	}
	if gen.shard > 1023 {
		t.Errorf("Auto-shard ID out of range: %d", gen.shard)
	}

	// Test with custom epoch
	customEpoch := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	gen, err = New(&Config{CustomEpochMs: customEpoch})
	if err != nil {
		t.Fatalf("New({CustomEpochMs: ...}) failed: %v", err)
	}
	if gen.baseEpoch != customEpoch {
		t.Errorf("Expected custom epoch, got default")
	}

	// Test with zero epoch
	gen, err = New(&Config{CustomEpochMs: 0})
	if err != nil {
		t.Fatalf("New({CustomEpochMs: 0}) failed: %v", err)
	}
	if gen.baseEpoch != defaultEpochMs {
		t.Errorf("Expected default epoch, got %d", gen.baseEpoch)
	}

	// Test auto-shard failure
	originalAutoShard := autoShardFunc
	autoShardFunc = func(d deps) (uint16, error) {
		return 0, errors.New("auto shard failed")
	}
	_, err = New(&Config{ShardID: -1})
	if err == nil {
		t.Error("Expected error from New with auto-shard failure, got nil")
	}
	autoShardFunc = originalAutoShard
}

// TestGenFunction tests the new Gen() wrapper function
func TestGenFunction(t *testing.T) {
	// Test case 1: Successful call with no config (uses default generator)
	id, err := Gen()
	if err != nil {
		t.Fatalf("Gen() failed: %v", err)
	}
	if len(id) != 11 {
		t.Errorf("Expected ID length 11 from Gen(), got %d", len(id))
	}

	// Test case 2: Successful call with a specific config
	id, err = Gen(&Config{ShardID: 42})
	if err != nil {
		t.Fatalf("Gen({ShardID: 42}) failed: %v", err)
	}
	if len(id) != 11 {
		t.Errorf("Expected ID length 11 from Gen({ShardID: 42}), got %d", len(id))
	}

	// Test case 3: Call with an invalid config
	_, err = Gen(&Config{ShardID: 9999})
	if err == nil {
		t.Error("Expected error for invalid shardID in Gen(), got nil")
	}

	// Test case 4: Test default generator initialization error
	defaultGenOnce = sync.Once{}
	defaultGen = nil
	defaultGenErr = nil

	originalNew := newFunc
	newFunc = func(cfg *Config) (*Generator, error) {
		return nil, errors.New("init failed")
	}

	_, err = Gen()
	if err == nil {
		t.Error("Expected error from failed default generator init, got nil")
	}
	if err.Error() != "init failed" {
		t.Errorf("Expected 'init failed' error, got '%v'", err)
	}

	newFunc = originalNew
	defaultGenOnce = sync.Once{}
	defaultGen = nil
	defaultGenErr = nil
}

// TestAutoShardLogic tests the different fallback paths for auto-sharding
func TestAutoShardLogic(t *testing.T) {
	// 1. MAC address based
	d1 := deps{
		ifacesFunc: func() ([]net.Interface, error) {
			return []net.Interface{{
				HardwareAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
			}}, nil
		},
	}
	shard1, err := autoShardWithDeps(d1)
	if err != nil {
		t.Fatalf("autoShardWithDeps(d1) failed: %v", err)
	}
	if shard1 == 0 {
		t.Error("Expected a non-zero shard from MAC address")
	}

	// 2. Hostname based (MAC fails)
	d2 := deps{
		ifacesFunc: func() ([]net.Interface, error) { return nil, errors.New("net error") },
		hostFunc:   func() (string, error) { return "test-host", nil },
	}
	shard2, err := autoShardWithDeps(d2)
	if err != nil {
		t.Fatalf("autoShardWithDeps(d2) failed: %v", err)
	}
	if shard2 == 0 {
		t.Error("Expected a non-zero shard from hostname")
	}

	// 3. Random bytes based (MAC and Hostname fail)
	d3 := deps{
		ifacesFunc: func() ([]net.Interface, error) { return nil, errors.New("net error") },
		hostFunc:   func() (string, error) { return "", errors.New("host error") },
		randFunc:   rand.Read,
	}
	shard3, err := autoShardWithDeps(d3)
	if err != nil {
		t.Fatalf("autoShardWithDeps(d3) failed: %v", err)
	}
	if shard3 == 0 {
		t.Error("Expected a non-zero shard from random bytes")
	}

	// 4. Error case (all others fail)
	d4 := deps{
		ifacesFunc: func() ([]net.Interface, error) { return nil, errors.New("net error") },
		hostFunc:   func() (string, error) { return "", errors.New("host error") },
		randFunc:   func(b []byte) (int, error) { return 0, errors.New("rand error") },
	}
	_, err = autoShardWithDeps(d4)
	if err == nil {
		t.Error("Expected error from autoShardWithDeps(d4), got nil")
	}

	// 5. Loopback interface should be skipped
	d5 := deps{
		ifacesFunc: func() ([]net.Interface, error) {
			return []net.Interface{
				{Flags: net.FlagLoopback, HardwareAddr: net.HardwareAddr{0xDE, 0xAD, 0xBE, 0xEF}},
				{HardwareAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x07}}, // This one should be used
			}, nil
		},
	}
	shard5, err := autoShardWithDeps(d5)
	if err != nil {
		t.Fatalf("autoShardWithDeps(d5) failed: %v", err)
	}
	if shard5 == 0 {
		t.Error("Expected a non-zero shard from the non-loopback MAC address")
	}

	// 6. Interface with no MAC address should be skipped
	d6 := deps{
		ifacesFunc: func() ([]net.Interface, error) {
			return []net.Interface{
				{HardwareAddr: nil},
				{HardwareAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x08}},
			}, nil
		},
	}
	shard6, err := autoShardWithDeps(d6)
	if err != nil {
		t.Fatalf("autoShardWithDeps(d6) failed: %v", err)
	}
	if shard6 == 0 {
		t.Error("Expected a non-zero shard from the valid second interface")
	}
}

// TestNextIDGeneration tests the Next() method
func TestNextIDGeneration(t *testing.T) {
	gen, _ := New(&Config{ShardID: 1})
	id := gen.Next()

	if len(id) != 11 {
		t.Errorf("Expected ID length 11, got %d", len(id))
	}

	// Test for uniqueness
	const numIDs = 10000
	idSet := make(map[string]struct{}, numIDs)
	for i := 0; i < numIDs; i++ {
		idSet[gen.Next()] = struct{}{}
	}
	if len(idSet) != numIDs {
		t.Errorf("Generated duplicate IDs, expected %d unique, got %d", numIDs, len(idSet))
	}
}

// TestSequenceRollover tests the sequence number rolling over
func TestSequenceRollover(t *testing.T) {
	mockTime := time.Now().UnixMilli()
	mockNowFunc := func() int64 {
		return mockTime
	}

	gen, _ := New(&Config{ShardID: 1})
	gen.deps.nowFunc = mockNowFunc

	// Exhaust the sequence
	for i := 0; i < 1<<15; i++ {
		_ = gen.Next()
	}

	// The next call should trigger the spin wait
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond) // Give the spin loop time to start
		mockTime++
	}()

	_ = gen.Next() // This will block until mockTime is incremented
	wg.Wait()

	if gen.lastMs != mockTime-gen.baseEpoch {
		t.Errorf("Expected lastMs to be updated after sequence rollover")
	}
	if gen.seq != 0 {
		t.Errorf("Expected sequence to be reset to 0 after rollover, got %d", gen.seq)
	}
}

// TestClockDrift tests handling of the system clock moving backwards
func TestClockDrift(t *testing.T) {
	mockTime := time.Now().UnixMilli()
	mockNowFunc := func() int64 {
		return mockTime
	}

	gen, _ := New(&Config{ShardID: 1})
	gen.deps.nowFunc = mockNowFunc

	_ = gen.Next()
	expectedLastMs := gen.lastMs

	// Move clock backwards
	mockTime--
	_ = gen.Next()

	if gen.lastMs != expectedLastMs {
		t.Errorf("lastMs should not decrease when clock moves back, expected %d, got %d", expectedLastMs, gen.lastMs)
	}
}

// TestSpinUntilNextMs tests the spin-wait mechanism
func TestSpinUntilNextMs(t *testing.T) {
	baseEpoch := int64(1000)
	lastMs := int64(500)
	now := lastMs + baseEpoch

	nowFunc := func() int64 {
		return now
	}

	go func() {
		time.Sleep(15 * time.Millisecond)
		now++
	}()

	spinUntilNextMs(baseEpoch, lastMs, nowFunc)

	if now-baseEpoch <= lastMs {
		t.Error("spinUntilNextMs returned before time advanced")
	}
}

func BenchmarkNextID(b *testing.B) {
	gen, _ := New(&Config{ShardID: 1})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.Next()
	}
}
