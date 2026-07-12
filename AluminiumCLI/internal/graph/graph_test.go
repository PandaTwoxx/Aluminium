package graph

import (
	"strings"
	"testing"
)

func TestParseSpec(t *testing.T) {
	tests := []struct {
		spec    string
		wantName string
		wantVer  string
	}{
		{"libpng@1.6.37", "libpng", "1.6.37"},
		{"zlib", "zlib", ""},
		{"@xyz", "", "xyz"},
	}

	for _, tt := range tests {
		gotName, gotVer := ParseSpec(tt.spec)
		if gotName != tt.wantName || gotVer != tt.wantVer {
			t.Errorf("ParseSpec(%q) = (%q, %q), want (%q, %q)", tt.spec, gotName, gotVer, tt.wantName, tt.wantVer)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"2.1", "2.0.9", 1},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.2.3", "1.2", 1},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.v1, tt.v2)
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
		}
	}
}

func TestTopoSort_NoCycle(t *testing.T) {
	// A -> B -> C
	g := DependencyGraph{
		"A": &Node{Name: "A", Dependencies: []string{"B"}},
		"B": &Node{Name: "B", Dependencies: []string{"C"}},
		"C": &Node{Name: "C", Dependencies: nil},
	}

	order, err := TopoSort(g, []string{"A"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantOrder := "C, B, A"
	gotOrder := strings.Join(order, ", ")
	if gotOrder != wantOrder {
		t.Errorf("TopoSort Order = %q, want %q", gotOrder, wantOrder)
	}
}

func TestTopoSort_Cycle(t *testing.T) {
	// A -> B -> C -> A
	g := DependencyGraph{
		"A": &Node{Name: "A", Dependencies: []string{"B"}},
		"B": &Node{Name: "B", Dependencies: []string{"C"}},
		"C": &Node{Name: "C", Dependencies: []string{"A"}},
	}

	_, err := TopoSort(g, []string{"A"})
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected error message to contain 'cycle', got: %v", err)
	}
}
