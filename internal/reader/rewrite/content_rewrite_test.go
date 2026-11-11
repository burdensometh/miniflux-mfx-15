// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rewrite // import "miniflux.app/v2/internal/reader/rewrite"

import (
	"strings"
	"testing"

	"miniflux.app/v2/internal/model"
)

// A rule whose selector matches nothing must leave the content byte-for-byte
// unchanged instead of replacing it with an empty serialization.
func TestApplyToSelectionNoMatchKeepsOriginalContent(t *testing.T) {
	content := `<p>Hello <strong>world</strong></p>`

	if got := addImageTitle(content); got != content {
		t.Errorf(`Expected content to be unchanged when no <img> is present, got %q`, got)
	}

	if got := removeTables(content); got != content {
		t.Errorf(`Expected content to be unchanged when no <table> is present, got %q`, got)
	}

	if got := addDynamicIframe(content); got != content {
		t.Errorf(`Expected content to be unchanged when no <iframe> is present, got %q`, got)
	}
}

// When a rule matches, the modification must be applied to the whole document,
// not just serialized from the first matched node.
func TestApplyToSelectionMatchKeepsWholeDocument(t *testing.T) {
	content := `<p>before</p><img src="a.png" alt="hello"><p>after</p>`
	got := addImageTitle(content)

	if !strings.Contains(got, `title="Hello"`) {
		t.Errorf(`Expected the rule to add a title attribute to the image, got %q`, got)
	}
	if !strings.Contains(got, `<p>before</p>`) || !strings.Contains(got, `<p>after</p>`) {
		t.Errorf(`Expected the rest of the document to be preserved, got %q`, got)
	}
}

// A rule that intentionally empties the content (e.g. removing the only
// element) is a match, not a no-op: the empty result must be kept and not
// "fixed" by restoring the original content.
func TestApplyToSelectionIntentionalEmptyResultIsKept(t *testing.T) {
	content := `<p>remove me</p>`

	if got := removeCustom(content, "p"); got != "" {
		t.Errorf(`Expected empty content after removing the only element, got %q`, got)
	}
}

func TestRemoveCustomKeepsUnmatchedContent(t *testing.T) {
	content := `<p>keep</p><div class="ad">drop</div><p>keep too</p>`
	got := removeCustom(content, ".ad")

	if strings.Contains(got, "drop") {
		t.Errorf(`Expected the matched element to be removed, got %q`, got)
	}
	if !strings.Contains(got, `<p>keep</p>`) || !strings.Contains(got, `<p>keep too</p>`) {
		t.Errorf(`Expected unmatched elements to be preserved, got %q`, got)
	}
}

// End-to-end through the public entry point: an entry whose content does not
// match any configured rule selector must come out with its content intact.
func TestApplyContentRewriteRulesNoMatchKeepsOriginalContent(t *testing.T) {
	entry := &model.Entry{
		URL:     "https://example.org/article",
		Title:   "Some article",
		Content: `<p>Hello <strong>world</strong></p>`,
	}

	ApplyContentRewriteRules(entry, "add_image_title\nremove_tables")

	if entry.Content != `<p>Hello <strong>world</strong></p>` {
		t.Errorf(`Expected entry content to be unchanged, got %q`, entry.Content)
	}
	if entry.Title != "Some article" {
		t.Errorf(`Expected entry title to be unchanged, got %q`, entry.Title)
	}
}
