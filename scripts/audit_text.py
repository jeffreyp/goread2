#!/usr/bin/env python3
"""
GoRead2 User-Facing Text Audit
Extracts all user-visible strings from HTML templates, JavaScript, and Go handlers.

Usage:
    python3 scripts/audit_text.py [--output report.md]
"""

import re
import sys
import os
from pathlib import Path
from html.parser import HTMLParser
from typing import NamedTuple

ROOT = Path(__file__).parent.parent


class TextEntry(NamedTuple):
    file: str
    line: int
    category: str
    text: str
    context: str = ""  # surrounding code or attribute name


def rel(path: Path) -> str:
    return str(path.relative_to(ROOT))


# ---------------------------------------------------------------------------
# HTML Template extractor
# ---------------------------------------------------------------------------

class HTMLTextExtractor(HTMLParser):
    """Extracts user-visible text from HTML templates."""

    # Attributes whose values are user-visible
    USER_ATTRS = {"placeholder", "title", "alt", "aria-label", "aria-placeholder",
                  "value", "data-tooltip"}

    # Tags whose text content is not user-visible (must have matching end tags)
    # Void elements like <meta> and <link> must NOT be here — they have no end tag
    # so decrementing on their close event never fires, corrupting the depth counter.
    SKIP_TAGS = {"script", "style", "head"}

    def __init__(self, filepath: str):
        super().__init__()
        self.filepath = filepath
        self.entries: list[TextEntry] = []
        self._skip_depth = 0
        self._current_tag = None

    def handle_starttag(self, tag, attrs):
        if tag in self.SKIP_TAGS:
            self._skip_depth += 1
            return
        self._current_tag = tag

        for attr, value in attrs:
            if attr in self.USER_ATTRS and value and value.strip():
                v = value.strip()
                if _is_meaningful(v):
                    self.entries.append(TextEntry(
                        file=self.filepath,
                        line=self.getpos()[0],
                        category=f"attribute: {attr}",
                        text=v,
                        context=f"<{tag} {attr}=...>",
                    ))

    def handle_endtag(self, tag):
        if tag in self.SKIP_TAGS:
            self._skip_depth -= 1

    def handle_data(self, data):
        if self._skip_depth > 0:
            return
        text = data.strip()
        if _is_meaningful(text):
            self.entries.append(TextEntry(
                file=self.filepath,
                line=self.getpos()[0],
                category="text content",
                text=text,
                context=f"<{self._current_tag or '?'}>",
            ))


def _is_meaningful(text: str) -> bool:
    """Return True if text is user-visible (not just whitespace, template vars, or punctuation)."""
    if len(text) < 2:
        return False
    # Pure template expression like {{ .Something }}
    if re.fullmatch(r'\{\{.*?\}\}', text.strip()):
        return False
    # Only symbols/numbers
    if re.fullmatch(r'[\d\s\W]+', text):
        return False
    return True


def extract_html(path: Path) -> list[TextEntry]:
    content = path.read_text(encoding="utf-8")
    extractor = HTMLTextExtractor(rel(path))
    extractor.feed(content)
    return extractor.entries


# ---------------------------------------------------------------------------
# JavaScript extractor
# ---------------------------------------------------------------------------

# Patterns for JS strings assigned to UI-relevant sinks
JS_UI_PATTERNS = [
    # textContent / innerText / innerHTML assignments
    (r"""\.(?:textContent|innerText)\s*=\s*[`'"](.+?)[`'"]""", "JS: textContent"),
    # innerHTML with literal strings (simple ones)
    (r"""\.innerHTML\s*=\s*[`'"](<[^'"`]+>)?([^'"`<]{4,})[`'"]""", "JS: innerHTML"),
    # showNotification / showToast / showError calls
    (r"""show(?:Notification|Toast|Error|Success|Warning)\s*\(\s*[`'"](.+?)[`'"]""", "JS: notification"),
    # showToast / toastManager calls
    (r"""toastManager\.show\s*\(\s*[`'"](.+?)[`'"]""", "JS: toast"),
    # confirm / alert / prompt dialogs
    (r"""(?:confirm|alert|prompt)\s*\(\s*[`'"](.+?)[`'"]""", "JS: dialog"),
    # throw new Error(...)
    (r"""throw\s+new\s+Error\s*\(\s*[`'"](.+?)[`'"]""", "JS: thrown error"),
    # console is NOT user-facing, skip
    # Fetch error handling: .message strings
    (r"""(?:error|err)\.message\s*\|\|\s*[`'"](.+?)[`'"]""", "JS: fallback error"),
    # Simple string passed to a function named like *message*, *label*, *title* etc.
    (r"""(?:message|label|title|text|placeholder)\s*[:=]\s*[`'"]([^'"`\n]{4,})[`'"]""", "JS: ui string"),
    # Button text / element creation
    (r"""createElement\([^)]+\).*?[`'"]([A-Z][^'"`\n]{3,})[`'"]""", "JS: element text"),
    # Template literal with significant text (non-HTML tags)
    (r"""`([^`\$\n]{10,})`""", "JS: template literal"),
]

# Patterns to SKIP (not user-visible)
JS_SKIP_PATTERNS = [
    r'^https?://',       # URLs
    r'^\.',              # CSS classes / selectors starting with .
    r'^#',              # IDs
    r'^\w+:',           # object keys like "key:"
    r'^[a-z-]+$',       # single lowercase identifiers (CSS class names, event names)
    r'^application/',   # MIME types
    r'^\s*$',
    r'^//.*',           # comments
]


def _js_meaningful(text: str) -> bool:
    text = text.strip()
    if len(text) < 3:
        return False
    for pat in JS_SKIP_PATTERNS:
        if re.match(pat, text):
            return False
    return True


def extract_js(path: Path) -> list[TextEntry]:
    entries = []
    lines = path.read_text(encoding="utf-8").splitlines()
    filepath = rel(path)

    for lineno, line in enumerate(lines, 1):
        # Skip comment lines
        stripped = line.strip()
        if stripped.startswith("//") or stripped.startswith("*"):
            continue

        seen_texts = set()
        for pattern, category in JS_UI_PATTERNS:
            for m in re.finditer(pattern, line):
                # Get last non-empty group
                text = next((g for g in reversed(m.groups()) if g), None)
                if text and _js_meaningful(text) and text not in seen_texts:
                    seen_texts.add(text)
                    entries.append(TextEntry(
                        file=filepath,
                        line=lineno,
                        category=category,
                        text=text.strip(),
                        context=stripped[:80],
                    ))

    return entries


# ---------------------------------------------------------------------------
# Go handler extractor
# ---------------------------------------------------------------------------

GO_USER_STRING_PATTERNS = [
    # gin.H{"error": "..."} or gin.H{"message": "..."}
    (r'''"(?:error|message|description|detail|title)"\s*:\s*"([^"]+)"''', "Go: API response"),
    # http.Error(w, "...", code)
    (r'''http\.Error\s*\([^,]+,\s*"([^"]+)"''', "Go: http.Error"),
    # fmt.Errorf("...") — may be user-visible
    (r'''fmt\.Errorf\s*\(\s*"([^"]+)"''', "Go: fmt.Errorf"),
    # errors.New("...")
    (r'''errors\.New\s*\(\s*"([^"]+)"''', "Go: errors.New"),
    # log messages with user content (fmt.Sprintf used for responses)
    (r'''fmt\.Sprintf\s*\(\s*"([^"]+)"''', "Go: fmt.Sprintf"),
]

GO_SKIP = [
    r'^%[svd]',      # pure format verbs
    r'^\w+$',        # single word (identifiers)
    r'^%',
]


def _go_meaningful(text: str) -> bool:
    text = text.strip()
    if len(text) < 4:
        return False
    for pat in GO_SKIP:
        if re.match(pat, text):
            return False
    return True


def extract_go(path: Path) -> list[TextEntry]:
    entries = []
    lines = path.read_text(encoding="utf-8").splitlines()
    filepath = rel(path)

    for lineno, line in enumerate(lines, 1):
        stripped = line.strip()
        if stripped.startswith("//"):
            continue

        seen_texts = set()
        for pattern, category in GO_USER_STRING_PATTERNS:
            for m in re.finditer(pattern, line):
                text = m.group(1)
                if text and _go_meaningful(text) and text not in seen_texts:
                    seen_texts.add(text)
                    entries.append(TextEntry(
                        file=filepath,
                        line=lineno,
                        category=category,
                        text=text.strip(),
                        context=stripped[:80],
                    ))

    return entries


# ---------------------------------------------------------------------------
# Report generator
# ---------------------------------------------------------------------------

def group_by_file(entries: list[TextEntry]) -> dict[str, list[TextEntry]]:
    groups: dict[str, list[TextEntry]] = {}
    for e in entries:
        groups.setdefault(e.file, []).append(e)
    return dict(sorted(groups.items()))


def render_markdown(entries: list[TextEntry]) -> str:
    lines = []
    lines.append("# GoRead2 User-Facing Text Audit\n")
    lines.append(f"_Total strings found: {len(entries)}_\n")
    lines.append(
        "This report captures all user-visible text in templates, JavaScript, and Go handlers.\n"
        "Use it to:\n"
        "- Identify tone/voice inconsistencies\n"
        "- Prepare strings for localization (i18n)\n"
        "- Spot redundant or confusing messages\n"
    )

    by_file = group_by_file(entries)

    for filepath, file_entries in by_file.items():
        lines.append(f"\n---\n\n## `{filepath}`\n")
        lines.append(f"_{len(file_entries)} string(s)_\n")
        lines.append("| Line | Category | Text | Context |")
        lines.append("|------|----------|------|---------|")
        for e in sorted(file_entries, key=lambda x: x.line):
            text = e.text.replace("|", "\\|").replace("\n", " ")
            context = e.context.replace("|", "\\|").replace("\n", " ")
            lines.append(f"| {e.line} | {e.category} | {text} | `{context}` |")

    return "\n".join(lines) + "\n"


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    output_path = None
    if "--output" in sys.argv:
        idx = sys.argv.index("--output")
        if idx + 1 < len(sys.argv):
            output_path = sys.argv[idx + 1]

    entries: list[TextEntry] = []

    # HTML templates
    for html_file in sorted((ROOT / "web" / "templates").glob("*.html")):
        entries.extend(extract_html(html_file))

    # JavaScript (non-minified)
    for js_file in sorted((ROOT / "web" / "static" / "js").glob("*.js")):
        if ".min." not in js_file.name:
            entries.extend(extract_js(js_file))

    # Go handlers
    for go_file in sorted((ROOT / "internal" / "handlers").glob("*.go")):
        if not go_file.name.endswith("_test.go"):
            entries.extend(extract_go(go_file))

    # Go services (errors.go has user-facing messages)
    for go_file in sorted((ROOT / "internal" / "services").glob("*.go")):
        if not go_file.name.endswith("_test.go"):
            entries.extend(extract_go(go_file))

    report = render_markdown(entries)

    if output_path:
        Path(output_path).write_text(report, encoding="utf-8")
        print(f"Report written to {output_path} ({len(entries)} strings)")
    else:
        print(report)


if __name__ == "__main__":
    main()
