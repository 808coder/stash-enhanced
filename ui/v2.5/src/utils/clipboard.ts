// Copies text to the clipboard. navigator.clipboard only exists in secure
// contexts (https or localhost); Stash is often served over plain http on a
// LAN, hence the fallback.
export function copyToClipboard(text: string) {
  if (navigator.clipboard) {
    navigator.clipboard.writeText(text).catch(() => {});
    return;
  }

  const el = document.createElement("textarea");
  el.value = text;
  el.style.position = "fixed";
  el.style.opacity = "0";
  document.body.appendChild(el);
  el.select();
  try {
    document.execCommand("copy");
  } catch {
    // nothing else to try
  }
  document.body.removeChild(el);
}
