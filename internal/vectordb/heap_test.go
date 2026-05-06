package vectordb

import "testing"

func TestCandidateHeap_MinOrder(t *testing.T) {
	h := newCandidateHeap(10)
	h.push(2, 5.0)
	h.push(1, 1.0)
	h.push(3, 3.0)

	n := h.popMin()
	if n.id != 1 {
		t.Fatalf("expected min id=1, got id=%d dist=%f", n.id, n.dist)
	}
}

func TestResultHeap_MaxOrder(t *testing.T) {
	h := newResultHeap(3)
	h.push(1, 1.0)
	h.push(2, 5.0)
	h.push(3, 3.0)
	h.push(4, 2.0)

	if h.worst() == 5.0 {
		t.Fatal("id=2 com dist=5.0 deveria ter sido removido")
	}
	if h.len() != 3 {
		t.Fatalf("expected len=3, got %d", h.len())
	}
}

func TestResultHeap_Worst(t *testing.T) {
	h := newResultHeap(2)
	if h.worst() != 1e38 {
		t.Fatal("empty heap worst must be 1e38")
	}
	h.push(1, 2.5)
	if h.worst() != 1e38 {
		t.Fatalf("underfull heap (1/2) must still return 1e38, got %f", h.worst())
	}
	h.push(2, 4.0)
	if h.worst() != 4.0 {
		t.Fatalf("full heap (2/2) worst must be 4.0, got %f", h.worst())
	}
	h.push(3, 1.0)
	if h.worst() != 2.5 {
		t.Fatalf("after replacing worst, expected 2.5, got %f", h.worst())
	}
}
