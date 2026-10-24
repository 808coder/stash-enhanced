import React, { useEffect, useMemo, useRef, useState } from "react";
import { Button, Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { ErrorMessage } from "src/components/Shared/ErrorMessage";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { getPlayer } from "src/components/ScenePlayer/util";
import { CaptionCue, languageMap, parseWebVTT } from "src/utils/caption";
import TextUtils from "src/utils/text";

type Caption = Pick<GQL.VideoCaption, "language_code" | "caption_type">;

interface ISceneCaptionsPanelProps {
  captionPath?: string | null;
  captions: Caption[];
  isVisible: boolean;
  onClickTimecode: (seconds: number) => void;
}

const captionKey = (c: Caption) => `${c.language_code}/${c.caption_type}`;

// returns the index of the last cue starting at or before time, or -1.
// cues must be sorted by start.
function findLastStartedCue(cues: CaptionCue[], time: number): number {
  let lo = 0;
  let hi = cues.length - 1;
  let found = -1;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    if (cues[mid].start <= time) {
      found = mid;
      lo = mid + 1;
    } else {
      hi = mid - 1;
    }
  }

  return found;
}

// returns the index of the cue containing time, or -1
function findActiveCue(cues: CaptionCue[], time: number): number {
  const i = findLastStartedCue(cues, time);
  return i !== -1 && time < cues[i].end ? i : -1;
}

// returns the index of the cue to scroll to for time: the active cue,
// or the next cue if time is in a gap. -1 if there is none.
function findScrollTargetCue(cues: CaptionCue[], time: number): number {
  const i = findActiveCue(cues, time);
  if (i !== -1) return i;

  const next = findLastStartedCue(cues, time) + 1;
  return next < cues.length ? next : -1;
}

// returns the nearest ancestor that scrolls itself, or undefined if only the
// page scrolls (e.g. the stacked mobile layout)
function getScrollContainer(el: HTMLElement): HTMLElement | undefined {
  for (let p = el.parentElement; p; p = p.parentElement) {
    if (p === document.body || p === document.documentElement) return;
    const { overflowY } = window.getComputedStyle(p);
    const scrollable = overflowY === "auto" || overflowY === "scroll";
    if (scrollable && p.scrollHeight > p.clientHeight) return p;
  }
}

// scrolls the cue with the given index to the centre of the panel's scroll
// container, unless it is already visible or filtered out
function scrollToCue(panel: HTMLElement, index: number) {
  const el = panel.querySelector<HTMLElement>(`[data-cue-index="${index}"]`);
  if (!el) return;

  const container = getScrollContainer(el);
  if (!container) return;

  const elRect = el.getBoundingClientRect();
  // not laid out, e.g. while the tab is still hidden
  if (elRect.height === 0) return;

  // leave room for the sticky controls
  const controls = panel.querySelector<HTMLElement>(".caption-controls");
  const containerRect = container.getBoundingClientRect();
  const top = containerRect.top + (controls?.offsetHeight ?? 0);
  const bottom = containerRect.bottom;

  if (elRect.top >= top && elRect.bottom <= bottom) return;

  const delta = elRect.top + elRect.height / 2 - (top + bottom) / 2;
  container.scrollBy({ top: delta, behavior: "smooth" });
}

export const SceneCaptionsPanel: React.FC<ISceneCaptionsPanelProps> = ({
  captionPath,
  captions,
  isVisible,
  onClickTimecode,
}) => {
  const intl = useIntl();

  const [selectedKey, setSelectedKey] = useState(captionKey(captions[0]));
  const [cache, setCache] = useState<Record<string, CaptionCue[]>>({});
  const [error, setError] = useState<string>();
  const [filter, setFilter] = useState("");
  const [activeIndex, setActiveIndex] = useState(-1);
  const panelRef = useRef<HTMLDivElement>(null);

  const selected =
    captions.find((c) => captionKey(c) === selectedKey) ?? captions[0];
  // primitives, so refetches of the scene don't restart the fetch effect
  const { language_code: lang, caption_type: type } = selected;
  const cues = cache[captionKey(selected)];

  function captionLabel(c: Caption) {
    const label = languageMap.get(c.language_code) ?? c.language_code;
    return `${label} (${c.caption_type})`;
  }

  // fetch the selected caption if not cached
  useEffect(() => {
    if (!isVisible || !captionPath || cues) return;

    const controller = new AbortController();
    const url = new URL(captionPath, window.location.href);
    url.searchParams.set("lang", lang);
    url.searchParams.set("type", type);
    const key = captionKey({ language_code: lang, caption_type: type });

    setError(undefined);
    fetch(url.toString(), { signal: controller.signal })
      .then((r) => {
        if (!r.ok) throw new Error(`${r.status} ${r.statusText}`);
        return r.text();
      })
      .then((text) => {
        setCache((prev) => ({ ...prev, [key]: parseWebVTT(text) }));
      })
      .catch((e) => {
        if (controller.signal.aborted) return;
        setError(e instanceof Error ? e.message : String(e));
      });

    return () => controller.abort();
  }, [isVisible, captionPath, lang, type, cues]);

  // track the active cue while the tab is active. Only the index is stored,
  // so the list re-renders when the active cue changes, not on every tick.
  // The list scrolls to the current position on seek and once on opening.
  useEffect(() => {
    if (!isVisible || !cues) return;

    let player: ReturnType<typeof getPlayer>;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let scrollFrame: number | undefined;
    const onTimeUpdate = () => {
      const time = player?.currentTime();
      setActiveIndex(time !== undefined ? findActiveCue(cues, time) : -1);
    };
    const scrollToCurrent = () => {
      const time = player?.currentTime();
      if (!panelRef.current || time === undefined) return;
      scrollToCue(panelRef.current, findScrollTargetCue(cues, time));
    };

    function bind() {
      player = getPlayer();
      if (!player) {
        // the player may not be ready yet
        retryTimer = setTimeout(bind, 500);
        return;
      }
      player.on("timeupdate", onTimeUpdate);
      player.on("seeked", scrollToCurrent);
      onTimeUpdate();
      if ((player.currentTime() ?? 0) > 0) {
        // wait for the tab to be laid out
        scrollFrame = requestAnimationFrame(scrollToCurrent);
      }
    }
    bind();

    return () => {
      clearTimeout(retryTimer);
      if (scrollFrame !== undefined) cancelAnimationFrame(scrollFrame);
      player?.off("timeupdate", onTimeUpdate);
      player?.off("seeked", scrollToCurrent);
    };
  }, [isVisible, cues]);

  const visibleCues = useMemo(() => {
    if (!cues) return [];
    const q = filter.trim().toLowerCase();
    return cues
      .map((cue, index) => ({ cue, index }))
      .filter(({ cue }) => !q || cue.text.toLowerCase().includes(q));
  }, [cues, filter]);

  function renderList() {
    if (error) {
      return (
        <ErrorMessage
          message={<FormattedMessage id="captions_panel.load_error" />}
          error={error}
        />
      );
    }
    if (!cues) return <LoadingIndicator />;
    if (cues.length === 0) {
      return <FormattedMessage id="captions_panel.no_captions" />;
    }
    if (visibleCues.length === 0) {
      return <FormattedMessage id="captions_panel.filter_no_matches" />;
    }

    return visibleCues.map(({ cue, index }) => (
      <div
        key={index}
        data-cue-index={index}
        className={`caption-cue${index === activeIndex ? " active" : ""}`}
      >
        <Button
          variant="link"
          className="caption-timecode"
          onClick={() => onClickTimecode(cue.start)}
        >
          {TextUtils.secondsToTimestamp(cue.start)}
        </Button>
        <div className="caption-text">{cue.text}</div>
      </div>
    ));
  }

  return (
    <div className="scene-captions-panel" ref={panelRef}>
      <div className="caption-controls mb-2 d-flex">
        {captions.length > 1 && (
          <Form.Control
            as="select"
            className="input-control mr-2 w-auto"
            value={captionKey(selected)}
            onChange={(e) => {
              setSelectedKey(e.target.value);
              setError(undefined);
            }}
          >
            {captions.map((c) => (
              <option key={captionKey(c)} value={captionKey(c)}>
                {captionLabel(c)}
              </option>
            ))}
          </Form.Control>
        )}
        <Form.Control
          className="text-input"
          value={filter}
          placeholder={intl.formatMessage({ id: "filter" })}
          onChange={(e) => setFilter(e.target.value)}
        />
      </div>
      {renderList()}
    </div>
  );
};

export default SceneCaptionsPanel;
