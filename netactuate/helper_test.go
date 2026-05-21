package netactuate

import "testing"

func TestSplitTags(t *testing.T) {
	got := splitTags(" kube, sjc, , cluster, kube ,SJC ")
	want := []string{"cluster", "kube", "sjc"}

	if len(got) != len(want) {
		t.Fatalf("expected %d tags, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tag %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func TestTagsStateFunc(t *testing.T) {
	got := tagsStateFunc(" kube, sjc, , cluster ")
	want := "cluster, kube, sjc"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSuppressTagsDiff(t *testing.T) {
	if !suppressTagsDiff("tags", "kube, sjc, cluster", "cluster, kube, sjc", nil) {
		t.Fatal("expected equivalent comma-separated tag sets to suppress diff")
	}
	if suppressTagsDiff("tags", "kube, sjc", "kube, ams", nil) {
		t.Fatal("expected different comma-separated tag sets to produce a diff")
	}
}
