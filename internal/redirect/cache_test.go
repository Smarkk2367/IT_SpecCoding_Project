package redirect

import "testing"

func TestCacheKey(t *testing.T) {
	if got := cacheKey("xK9mP"); got != "redirect:xK9mP" {
		t.Fatalf("expected redirect:xK9mP, got %s", got)
	}
}
