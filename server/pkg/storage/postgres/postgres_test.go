package postgres

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"xtestserver/domain"
)

const tdbEnv = "RATES_TEST_DB"

var tdb *Postgres

func restoreDB(tdb *Postgres) error {
	b, err := os.ReadFile(filepath.Join("testdata", "t.sql"))
	if err != nil {
		return err
	}

	return tdb.exec(context.Background(), string(b))
}

func TestMain(m *testing.M) {

	connstr, ok := os.LookupEnv(tdbEnv)
	if !ok {
		fmt.Fprintf(os.Stderr, "environment variable %q must be set\n", tdbEnv)
		os.Exit(m.Run()) // tests will be skipped
	}

	var err error
	tdb, err = New(connstr)
	if err != nil {
		log.Fatalf("db connection: %v", err)
	}
	defer tdb.Close()

	if err = restoreDB(tdb); err != nil {
		tdb.Close()
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func TestPostgres(t *testing.T) {
	if tdb == nil {
		t.Skip("no connection to test database, skipped...")
	}

	t.Run("AddBtcRate", func(t *testing.T) {
		want := testBtcRate3

		err := tdb.AddBtcRate(context.Background(), want)
		if err != nil {
			t.Fatalf("AddBtcRate() = error: %v", err)
		}

		got, err := tdb.BtcRate(context.Background(), 1, 2)
		if err != nil {
			t.Fatalf("BtcRate() = error: %v", err)
		}

		if len(got) != 1 {
			t.Fatalf("BtcRate() rows = %d, want %d", len(got), 1)
		}

		if got[0] != want {
			t.Errorf("AddBtcRate() = %#v, want %#v", got, want)
		}

	})

	t.Run("BtcRate", func(t *testing.T) {
		want := testBtcRate1

		got, err := tdb.BtcRate(context.Background(), 1, 0)
		if err != nil {
			t.Fatalf("BtcRate() = error: %v", err)
		}

		if len(got) != 1 {
			t.Fatalf("BtcRate() rows = %d, want %d", len(got), 1)
		}

		if got[0] != want {
			t.Errorf("BtcRate() = %#v, want %#v", got, want)
		}

	})

	t.Run("AddFiats", func(t *testing.T) {
		wantFiats := []domain.Rate{
			testFiatRate1, testFiatRate2, testFiatRate3, testFiatRate4}

		err := tdb.AddFiats(context.Background(), []domain.Rate{testFiatRate3, testFiatRate4}...)
		if err != nil {
			t.Fatalf("AddFiats() = error: %v", err)
		}

		gotFiats, err := tdb.FiatsCurrent(context.Background())
		if err != nil {
			t.Fatalf("FiatsCurrent() = error: %v", err)
		}

		if len(gotFiats) != 4 {
			t.Fatalf("FiatsCurrent() rows = %d, want %d", len(gotFiats), 4)
		}

		for i := range gotFiats {
			if gotFiats[i] != wantFiats[i] {
				t.Errorf("AddFiats() = %#v, want %#v", gotFiats[i], wantFiats[i])
			}
		}

	})

	t.Run("RUBUSDRate", func(t *testing.T) {
		want := testFiatRate1

		got, err := tdb.RUBUSDRate(context.Background())
		if err != nil {
			t.Fatalf("RUBUSDRate() = error: %v", err)
		}

		if got != want {
			t.Errorf("RUBUSDRate() = %#v, want %#v", got, want)
		}
	})
}

var testBtcRate1 = domain.Rate{Id: 1, Time: 1658252361, Value: 22278.20}
var testBtcRate2 = domain.Rate{Id: 2, Time: 1658252362, Value: 22378.20}
var testBtcRate3 = domain.Rate{Id: 3, Time: 1658252363, Value: 11111.10}

// var testFiat1 = domain.Rate{Id: 1, CharCode: "USD", Nominal: 1, Time: 1658252361, Value: 22278.20}
// var testFiat2 = domain.Rate{Id: 2, CharCode: "HUF", Nominal: 100, Time: 1658252361, Value: 22278.20}
var testFiatRate1 = domain.Rate{Id: 1, CharCode: "USD", Nominal: 1, Time: 1658252361, Value: 22278.20}
var testFiatRate2 = domain.Rate{Id: 2, CharCode: "HUF", Nominal: 100, Time: 1658252361, Value: 32378.20}
var testFiatRate3 = domain.Rate{Id: 3, CharCode: "AZN", Nominal: 1, Time: 1658314796, Value: 5555555555.2024}
var testFiatRate4 = domain.Rate{Id: 4, CharCode: "GBP", Nominal: 1, Time: 1658314796, Value: 4444444444.3334}
