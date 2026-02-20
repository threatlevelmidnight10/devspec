package gitutil

import "testing"

func TestDiffLineCount(t *testing.T) {
	diff := `diff --git a/a.txt b/a.txt
index 123..456 100644
--- a/a.txt
+++ b/a.txt
@@ -1,2 +1,2 @@
-line one
+line two
 context`

	got := DiffLineCount(diff)
	if got != 2 {
		t.Fatalf("expected 2 changed lines, got %d", got)
	}
}
