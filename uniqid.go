package uniqid

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"hash/fnv"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
const defaultEpochMs = int64(1577836800000) // 2020-01-01

// Config defines options for creating a Generator.
//
// Fields:
//   - ShardID: Node identifier [0..1023]. Use -1 to auto-detect.
//   - CustomEpochMs: Custom epoch in milliseconds (default = Unix epoch).
type Config struct {
	ShardID       int
	CustomEpochMs int64
}

// Generator produces unique, time-sortable IDs.
// It is safe for concurrent use by multiple goroutines.
type Generator struct {
	mu        sync.Mutex
	lastMs    int64
	seq       uint16
	shard     uint16
	baseEpoch int64
	deps      deps
}

var autoShardFunc = autoShardWithDeps

// New creates a new ID generator with the given configuration.
//
// Config options:
//   - ShardID (int):
//     The node/shard identifier. Must be in the range [0, 1023].
//     If set to -1, the shard ID will be auto-derived from
//     network interface, hostname, or randomness.
//   - CustomEpochMs (int64):
//     Custom epoch timestamp in milliseconds (default is Unix epoch).
//     Useful if you want to shorten IDs by moving the epoch closer
//     to the present time.
//
// Example:
//
//	gen, err := uniqid.New(&uniqid.Config{
//	    ShardID:       1,
//	    CustomEpochMs: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	id := gen.Next()
//	fmt.Println(id) // Example: "Ab3Xyz0LmN_"
//
// If cfg is nil, defaults are used (auto shard ID, epoch = 1970).
func New(cfg *Config) (*Generator, error) {
	if cfg == nil {
		cfg = &Config{ShardID: -1, CustomEpochMs: defaultEpochMs}
	}
	epoch := cfg.CustomEpochMs
	if epoch == 0 {
		epoch = defaultEpochMs
	}

	g := &Generator{
		baseEpoch: epoch,
		deps: deps{
			nowFunc:    func() int64 { return time.Now().UnixMilli() },
			ifacesFunc: net.Interfaces,
			hostFunc:   os.Hostname,
			randFunc:   rand.Read,
		},
	}

	if cfg.ShardID >= 0 {
		if cfg.ShardID > 1023 {
			return nil, errors.New("shardID must be 0..1023")
		}
		g.shard = uint16(cfg.ShardID)
	} else {
		shard, err := autoShardFunc(g.deps)
		if err != nil {
			return nil, err
		}
		g.shard = shard
	}

	return g, nil
}

var (
	defaultGen     *Generator
	defaultGenErr  error
	defaultGenOnce sync.Once
)

var newFunc = New

// Gen creates a new unique ID, optionally with a custom configuration.
//
// If no config is provided, it uses a default, package-level generator.
// This is the simplest way to get a unique ID.
// Example:
//
//	id, err := uniqid.Gen()
//
// If a config is provided, it creates a new generator for this specific
// call. This is useful for one-off ID generation with special settings.
// Example:
//
//	id, err := uniqid.Gen(&uniqid.Config{ShardID: 2})
func Gen(cfgs ...*Config) (string, error) {
	if len(cfgs) == 0 {
		defaultGenOnce.Do(func() {
			defaultGen, defaultGenErr = newFunc(nil)
		})
		if defaultGenErr != nil {
			return "", defaultGenErr
		}
		return defaultGen.Next(), nil
	}
	g, err := newFunc(cfgs[0])
	if err != nil {
		return "", err
	}
	return g.Next(), nil
}

// Next generates a new unique 11-character ID.
// IDs are:
//   - Time-sortable (monotonic)
//   - Collision-free (with 15-bit sequence per millisecond)
//   - Shard-aware (10-bit shard ID)
//
// Example output: "Ab3Xyz0LmN_"
func (g *Generator) Next() string {
	g.mu.Lock()
	nowMs := max(g.deps.nowFunc()-g.baseEpoch, g.lastMs)
	if nowMs == g.lastMs {
		g.seq++
		if g.seq >= 1<<15 {
			g.mu.Unlock()
			spinUntilNextMs(g.baseEpoch, nowMs, g.deps.nowFunc)
			g.mu.Lock()
			nowMs = g.deps.nowFunc() - g.baseEpoch
			g.lastMs = nowMs
			g.seq = 0
		}
	} else {
		g.seq = 0
		g.lastMs = nowMs
	}
	val := (uint64(nowMs) << 25) | (uint64(g.shard) << 15) | uint64(g.seq)
	g.mu.Unlock()
	var out [11]byte
	for i := 10; i >= 0; i-- {
		out[i] = alphabet[val&63]
		val >>= 6
	}
	return string(out[:])
}

// -------------------------------------------------------------------
// Internal helpers (not exported, used for testing & implementation).
// -------------------------------------------------------------------

// deps groups system dependencies (time, network, randomness).
// Used internally for testing by injecting fake implementations.
// Not exported; normal users of the library never touch this.
type deps struct {
	nowFunc    func() int64
	ifacesFunc func() ([]net.Interface, error)
	hostFunc   func() (string, error)
	randFunc   func([]byte) (int, error)
}

// autoShardWithDeps tries to derive a shard ID automatically from
// network interface MAC, hostname, or random fallback.
// Used internally when Config.ShardID = -1.
func autoShardWithDeps(d deps) (uint16, error) {
	if ifs, _ := d.ifacesFunc(); len(ifs) > 0 {
		for _, in := range ifs {
			if in.Flags&net.FlagLoopback != 0 || len(in.HardwareAddr) == 0 {
				continue
			}
			h := fnv.New32a()
			_, _ = h.Write(in.HardwareAddr)
			return uint16(h.Sum32() & 0x3FF), nil
		}
	}
	if hn, err := d.hostFunc(); err == nil {
		h := fnv.New32a()
		_, _ = h.Write([]byte(hn))
		return uint16(h.Sum32() & 0x3FF), nil
	}
	var b [2]byte
	if _, err := d.randFunc(b[:]); err == nil {
		return binary.BigEndian.Uint16(b[:]) & 0x3FF, nil
	}
	return 0, errors.New("could not determine shard ID")
}

// spinUntilNextMs blocks until the next millisecond tick.
// Used to ensure monotonic IDs when the per-ms counter overflows.
// Not exported.
func spinUntilNextMs(baseEpoch, lastMs int64, nowFunc func() int64) {
	target := lastMs + 1
	for {
		now := nowFunc() - baseEpoch
		if now >= target {
			return
		}
		runtime.Gosched()
		time.Sleep(10 * time.Microsecond)
	}
}
