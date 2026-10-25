export const languageMap = new Map<string, string>([
  ["de", "Deutsche"],
  ["en", "English"],
  ["es", "Español"],
  ["fr", "Français"],
  ["it", "Italiano"],
  ["ja", "日本"],
  ["ko", "한국인"],
  ["nl", "Holandés"],
  ["pt", "Português"],
  ["ru", "Русский"],
  ["00", "Unknown"], // stash reserved language code
]);

export const valueToCode = (value?: string | null) => {
  if (!value) {
    return undefined;
  }

  return Array.from(languageMap.keys()).find((v) => {
    return languageMap.get(v) === value;
  });
};

export interface CaptionCue {
  start: number;
  end: number;
  text: string;
}

// parses a WebVTT timestamp (mm:ss.ttt or hh:mm:ss.ttt) into seconds
function parseVTTTimestamp(value: string): number | undefined {
  const match = /^(?:(\d+):)?(\d{1,2}):(\d{2})[.,](\d{1,3})$/.exec(value);
  if (!match) return undefined;

  const [, h, m, s, ms] = match;
  return (
    Number(h ?? 0) * 3600 +
    Number(m) * 60 +
    Number(s) +
    Number(ms.padEnd(3, "0")) / 1000
  );
}

const entities: Record<string, string> = {
  "&amp;": "&",
  "&lt;": "<",
  "&gt;": ">",
  "&nbsp;": " ",
};

function cleanCueText(text: string): string {
  return text
    .replace(/\{\\[^}]*\}/g, "") // override tags, e.g. {\an8}
    .replace(/<[^>]*>/g, "") // <i>, <c.x>, <00:01.000>, etc.
    .replace(/&(?:amp|lt|gt|nbsp);/g, (e) => entities[e])
    .trim();
}

// parses WebVTT text into a list of cues, sorted by start time
export function parseWebVTT(text: string): CaptionCue[] {
  const cues: CaptionCue[] = [];
  const blocks = text
    .replace(/^﻿/, "")
    .replace(/\r\n?/g, "\n")
    .split(/\n{2,}/);

  for (const block of blocks) {
    const lines = block.split("\n");
    const timingIndex = lines.findIndex((l) => l.includes("-->"));
    if (timingIndex === -1 || timingIndex > 1) continue;

    // skip blocks that are not cues
    if (/^(NOTE|STYLE|REGION)(\s|$)/.test(lines[0])) continue;

    const [startStr, rest] = lines[timingIndex].split("-->");
    const endStr = rest?.trim().split(/\s+/)[0];
    const start = parseVTTTimestamp(startStr.trim());
    const end = endStr ? parseVTTTimestamp(endStr) : undefined;
    if (start === undefined || end === undefined) continue;

    const cueText = cleanCueText(lines.slice(timingIndex + 1).join("\n"));
    if (!cueText) continue;

    cues.push({ start, end, text: cueText });
  }

  return cues.sort((a, b) => a.start - b.start);
}
