package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestSaveMake_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveMake(context.Background(), Make{ID: "toyota", Name: "Toyota"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sess.closed {
		t.Fatal("session not closed")
	}
}

func TestSaveMake_Error(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveMake(context.Background(), Make{ID: "toyota", Name: "Toyota"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSaveVehicleModel_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveVehicleModel(context.Background(), VehicleModel{ID: "toyota-camry", Name: "Camry", MakeID: "toyota"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveGeneration_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveGeneration(context.Background(), Generation{ID: "gen1", Name: "XV70", Platform: "TNGA-K", StartYear: 2018, EndYear: 2024, ModelID: "toyota-camry"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveTrim_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveTrim(context.Background(), Trim{ID: "t1", Name: "SE", Engine: "2.5L", Transmission: "8AT", Drivetrain: "FWD", GenerationID: "gen1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveModelYear_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveModelYear(context.Background(), ModelYear{ID: "toyota-camry-2020", Year: 2020, Make: "Toyota", Model: "Camry"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveSystem_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveSystem(context.Background(), System{ID: "engine", Name: "Engine"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveSubsystem_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.SaveSubsystem(context.Background(), Subsystem{ID: "fuel-injection", Name: "Fuel Injection", SystemID: "engine"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureVehicleHierarchy_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.EnsureVehicleHierarchy(context.Background(), VehicleInfo{Make: "Toyota", Model: "Camry", Year: 2020})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureVehicleHierarchy_TxError(t *testing.T) {
	sess := &mockSessionWithTxErr{err: errors.New("tx fail")}
	gs := NewWithOpener(&mockOpener2{session: sess})

	err := gs.EnsureVehicleHierarchy(context.Background(), VehicleInfo{Make: "Toyota", Model: "Camry", Year: 2020})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLinkComponentToVehicle_Success(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.LinkComponentToVehicle(context.Background(), "comp1", "toyota-camry-2020")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkComponentToVehicle_Error(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	err := gs.LinkComponentToVehicle(context.Background(), "comp1", "toyota-camry-2020")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindComponentsByVehicle_Success(t *testing.T) {
	records := []*neo4j.Record{
		makeNodeRecord(map[string]any{"id": "c1", "name": "Alternator", "type": "component", "vehicle": "v1"}),
	}
	sess := &mockSession{runResult: newMockResult(records...)}
	gs := NewWithOpener(&mockOpener{session: sess})

	comps, err := gs.FindComponentsByVehicle(context.Background(), VehicleInfo{Make: "Toyota", Model: "Camry", Year: 2020})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
}

func TestFindComponentsByVehicle_Error(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, err := gs.FindComponentsByVehicle(context.Background(), VehicleInfo{Make: "Toyota", Model: "Camry", Year: 2020})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetVehicleHierarchy_NotFound(t *testing.T) {
	sess := &mockSession{runResult: newMockResult()}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, _, _, err := gs.GetVehicleHierarchy(context.Background(), VehicleInfo{Make: "Toyota", Model: "Camry", Year: 2020})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestGetVehicleHierarchy_Error(t *testing.T) {
	sess := &mockSession{runErr: errors.New("fail")}
	gs := NewWithOpener(&mockOpener{session: sess})

	_, _, _, err := gs.GetVehicleHierarchy(context.Background(), VehicleInfo{Make: "Toyota", Model: "Camry", Year: 2020})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStrFromNode_Map(t *testing.T) {
	val := map[string]any{"id": "test", "name": "Test"}
	if got := strFromNode(val, "id"); got != "test" {
		t.Fatalf("expected test, got %s", got)
	}
}

func TestStrFromNode_Nil(t *testing.T) {
	if got := strFromNode(nil, "id"); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestIntFromNode_Map(t *testing.T) {
	val := map[string]any{"year": 2020}
	if got := intFromNode(val, "year"); got != 2020 {
		t.Fatalf("expected 2020, got %d", got)
	}
}

func TestIntFromNode_Int64(t *testing.T) {
	val := map[string]any{"year": int64(2021)}
	if got := intFromNode(val, "year"); got != 2021 {
		t.Fatalf("expected 2021, got %d", got)
	}
}

func TestIntFromNode_Float64(t *testing.T) {
	val := map[string]any{"year": float64(2022)}
	if got := intFromNode(val, "year"); got != 2022 {
		t.Fatalf("expected 2022, got %d", got)
	}
}

func TestIntFromNode_Nil(t *testing.T) {
	if got := intFromNode(nil, "year"); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}
