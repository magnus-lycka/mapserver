package settings

import (
	"io/ioutil"
	"mapserver/mapobjectdb/sqlite"
	"mapserver/types"
	"os"
	"testing"
)

func TestStrings(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "TileDBTest.*.sqlite")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	db, err := sqlite.New(tmpfile.Name())
	if err != nil {
		panic(err)
	}

	err = db.Migrate()
	if err != nil {
		panic(err)
	}

	s := New(db)

	//string

	s.SetString("k", "v")
	str := s.GetString("k", "v2")
	if str != "v" {
		t.Fatal("getstring failed: " + str)
	}

	if s.GetString("k2", "v3") != "v3" {
		t.Fatal("getstring with default failed")
	}

	//int

	s.SetInt("i", 123)
	i := s.GetInt("i", 456)
	if i != 123 {
		t.Fatal("getint failed")
	}

	s.SetInt("i3", -123)
	i = s.GetInt("i3", 456)
	if i != -123 {
		t.Fatal("getint negative failed")
	}

	if s.GetInt("i2", 111) != 111 {
		t.Fatal("getint with default failed")
	}

	//int64

	s.SetInt64("i", 1230000012300056)
	i2 := s.GetInt64("i", 456)
	if i2 != 1230000012300056 {
		t.Fatal("getint64 failed")
	}

	if s.GetInt64("i2", 12300000123000564) != 12300000123000564 {
		t.Fatal("getint with default failed")
	}

	//bool

	s.SetBool("b", false)
	b2 := s.GetBool("b", true)
	if b2 {
		t.Fatal("getbool failed")
	}

	if s.GetBool("b2", false) {
		t.Fatal("getbool with default failed")
	}

	if !s.GetBool("b2", true) {
		t.Fatal("getbool with default failed")
	}

}

func TestResetInitialRender(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "ResetInitialRender.*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	db, err := sqlite.New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}
	s := New(db)
	layers := []*types.Layer{{Id: 7}, {Id: -3}, {Id: 2}}

	if err := ResetInitialRender(s, layers); err != nil {
		t.Fatal(err)
	}
	if !s.GetBool(SETTING_INITIAL_RUN, false) {
		t.Fatal("initial run was not enabled")
	}
	if got := s.GetInt(SETTING_LAST_LAYER, 0); got != -3 {
		t.Fatalf("last layer=%d, want -3", got)
	}
	if got := s.GetInt(SETTING_LAST_X_BLOCK, 0); got != -129 {
		t.Fatalf("last x=%d", got)
	}
	if got := s.GetInt(SETTING_LAST_Y_BLOCK, 0); got != -128 {
		t.Fatalf("last y=%d", got)
	}
	if got := s.GetInt64(SETTING_TOTAL_LEGACY_COUNT, 0); got != -1 {
		t.Fatalf("legacy total=%d", got)
	}
	if got := s.GetInt64(SETTING_PROCESSED_LEGACY_COUNT, -1); got != 0 {
		t.Fatalf("legacy processed=%d", got)
	}
}
