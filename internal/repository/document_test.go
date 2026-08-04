package repository

import "testing"

func TestDocumentDao_ClaimPendingDocuments_RejectsNonPositiveLimit(t *testing.T) {
	dao := &DocumentDao{}

	docs, err := dao.ClaimPendingDocuments(0)
	if err != nil {
		t.Fatalf("ClaimPendingDocuments(0) error = %v, want nil", err)
	}
	if docs != nil {
		t.Fatalf("ClaimPendingDocuments(0) docs = %#v, want nil", docs)
	}
}
