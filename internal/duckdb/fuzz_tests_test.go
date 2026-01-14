package duckdb_test

import (
	"bytes"
	"errors"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"cogni/internal/duckdb"
	"cogni/internal/testutil"
)

// TestCanonicalJSONFuzzStability validates canonical JSON ordering across random specs.
func TestCanonicalJSONFuzzStability(t *testing.T) {
	ctx := testutil.Context(t, 5*time.Second)
	runWithTimeout(t, ctx, func() error {
		cfg := &quick.Config{MaxCount: 200, Rand: rand.New(rand.NewSource(42))}
		property := func(seed int64) bool {
			rng := rand.New(rand.NewSource(seed))
			spec := randomSpec(rng, maxSpecDepth)
			left, err := duckdb.CanonicalJSON(spec)
			if err != nil {
				return false
			}
			rehydrated, err := rehydrateSpec(spec)
			if err != nil {
				return false
			}
			right, err := duckdb.CanonicalJSON(rehydrated)
			if err != nil {
				return false
			}
			return bytes.Equal(left, right)
		}
		if err := quick.Check(property, cfg); err != nil {
			if seed, ok := extractSeed(err); ok {
				rng := rand.New(rand.NewSource(seed))
				spec := randomSpec(rng, maxSpecDepth)
				_ = writeSeedFixture("canonical-json", seed, spec)
			}
			return err
		}
		return nil
	})
}

// TestFingerprintCollisionFuzz checks for hash collisions over random specs.
func TestFingerprintCollisionFuzz(t *testing.T) {
	ctx := testutil.Context(t, 5*time.Second)
	runWithTimeout(t, ctx, func() error {
		seen := make(map[string]string)
		for i := 0; i < 300; i++ {
			seed := int64(i + 1)
			rng := rand.New(rand.NewSource(seed))
			spec := randomSpec(rng, maxSpecDepth)
			canonical, err := duckdb.CanonicalJSON(spec)
			if err != nil {
				return err
			}
			fp, err := duckdb.FingerprintJSON(spec)
			if err != nil {
				return err
			}
			if prevCanonical, ok := seen[fp]; ok {
				if prevCanonical == string(canonical) {
					continue
				}
				payload := map[string]interface{}{
					"collision_seed": seed,
					"current_spec":   spec,
				}
				_ = writeSeedFixture("fingerprint-collision", seed, payload)
				return errors.New("fingerprint collision detected")
			}
			seen[fp] = string(canonical)
		}
		return nil
	})
}

// extractSeed extracts the failing seed from a quick.Check error.
func extractSeed(err error) (int64, bool) {
	var checkErr *quick.CheckError
	if !errors.As(err, &checkErr) {
		return 0, false
	}
	if len(checkErr.In) == 0 {
		return 0, false
	}
	switch v := checkErr.In[0].(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case reflect.Value:
		if v.Kind() == reflect.Int64 {
			return v.Int(), true
		}
		if v.Kind() == reflect.Int {
			return int64(v.Int()), true
		}
	default:
		return 0, false
	}
	return 0, false
}
