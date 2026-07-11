// PubScale offer copy (description, goal instructions) sometimes embeds
// literal "<br>" tags and Unicode line-separator characters (U+2028/U+2029)
// instead of real newlines — rendering it as plain text shows the literal
// "<br>" on screen instead of a line break.
//
// We deliberately never use dangerouslySetInnerHTML here (this is
// untrusted third-party advertiser copy) — instead we split on the known
// break markers and render real <br/> elements between plain-text nodes.
const BREAK_PATTERN = /<br\s*\/?>|\u2028|\u2029/gi;

/** Renders text with any <br>/line-separator markers turned into real line breaks. */
export function RichText({ text, className }: { text: string; className?: string }) {
  const lines = text.split(BREAK_PATTERN);
  return (
    <span className={className}>
      {lines.map((line, i) => (
        <span key={i}>
          {line}
          {i < lines.length - 1 && <br />}
        </span>
      ))}
    </span>
  );
}

/** Collapses break markers into spaces for single-line/line-clamped previews (e.g. offer cards). */
export function flattenText(text: string): string {
  return text.replace(BREAK_PATTERN, " ").replace(/\s+/g, " ").trim();
}
