// Accept a UID, an API resource name, or a canonical link from this instance.
export function storyMemoName(input: string, origin: string): string | undefined {
  let value = input.trim();
  if (!value) return "";
  if (/^https?:\/\//i.test(value)) {
    try {
      const url = new URL(value);
      if (url.origin !== origin) return undefined;
      value = url.pathname;
    } catch {
      return undefined;
    }
  }
  value = value.replace(/^\/?memos\//, "").replace(/\/$/, "");
  return /^[a-zA-Z0-9_-]+$/.test(value) ? `memos/${value}` : undefined;
}
