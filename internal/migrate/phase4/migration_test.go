package phase4migrate

import "testing"

func TestPhase4MigrationVersionsAdvanceGlobalDataVersion(t *testing.T) {
	if len(versionMigrations) != 2 {
		t.Fatalf("versionMigrations len = %d, want 2", len(versionMigrations))
	}

	if versionMigrations[0].Version != "0.0.5" {
		t.Fatalf("first migration version = %q, want %q", versionMigrations[0].Version, "0.0.5")
	}
	if versionMigrations[1].Version != "0.0.6" {
		t.Fatalf("second migration version = %q, want %q", versionMigrations[1].Version, "0.0.6")
	}
}
