package repo

import (
	"context"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Test the neo4jSessionAdapter wrapping a real neo4j session interface.
// Since we can't easily mock neo4j.SessionWithContext, we test the adapter
// methods exist and the session() fallback path.

type fakeDriver struct {
	neo4j.DriverWithContext
	sessionCreated bool
}

type fakeSession struct {
	neo4j.SessionWithContext
}

func (d *fakeDriver) NewSession(_ context.Context, _ neo4j.SessionConfig) neo4j.SessionWithContext {
	d.sessionCreated = true
	return &fakeSession{}
}

// TestSession_NilNewSession tests the fallback path where newSession is nil.
func TestSession_NilNewSession(t *testing.T) {
	fd := &fakeDriver{}
	r := &Neo4jRepo[string, string]{
		driver: fd,
	}
	// newSession is nil, should use driver.NewSession
	sess := r.session(context.Background())
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
	if !fd.sessionCreated {
		t.Fatal("expected driver.NewSession to be called")
	}

	// Verify it's a neo4jSessionAdapter
	adapter, ok := sess.(*neo4jSessionAdapter)
	if !ok {
		t.Fatal("expected neo4jSessionAdapter")
	}
	_ = adapter
}

// TestNeo4jSessionAdapter_RunAndClose tests the adapter methods indirectly.
// We can't call them without a real session, but we verify the types compile.
func TestNeo4jSessionAdapter_Compile(t *testing.T) {
	// Just verify the interface is satisfied
	var _ runner = (*neo4jSessionAdapter)(nil)
}
