import React, { useEffect, useRef, useState } from "react";
import { Badge, Button } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { Link } from "react-router-dom";
import cx from "classnames";
import {
  faCheck,
  faChevronDown,
  faChevronRight,
  faCopy,
} from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { useConfigurationContext } from "src/hooks/Config";
import { copyToClipboard } from "src/utils/clipboard";
import NavUtils from "src/utils/navigation";
import {
  convertToRatingFormat,
  defaultRatingSystemOptions,
  RatingSystemType,
} from "src/utils/rating";
import { getStashboxBase } from "src/utils/stashbox";
import TextUtils from "src/utils/text";
import { ExternalLink } from "../Shared/ExternalLink";
import { Icon } from "../Shared/Icon";
import { TombstoneFileSize } from "./TombstoneFileSize";
import { TombstoneData, TombstoneFile, TombstonePerformer } from "./types";

// "BASENAME" is shown to users as "Filename"
const matchKeyMessageIDs: Record<GQL.TombstoneMatchKey, string> = {
  [GQL.TombstoneMatchKey.Url]: "tombstones.match_keys.url",
  [GQL.TombstoneMatchKey.StashId]: "tombstones.match_keys.stash_id",
  [GQL.TombstoneMatchKey.Oshash]: "tombstones.match_keys.oshash",
  [GQL.TombstoneMatchKey.Md5]: "tombstones.match_keys.md5",
  [GQL.TombstoneMatchKey.Phash]: "tombstones.match_keys.phash",
  [GQL.TombstoneMatchKey.Code]: "tombstones.match_keys.code",
  [GQL.TombstoneMatchKey.Title]: "tombstones.match_keys.title",
  [GQL.TombstoneMatchKey.Basename]: "tombstones.match_keys.basename",
};

// The frozen gender, else the gender of the matching library performer.
// Unknown gender counts as non-male.
function isMale(p: TombstonePerformer) {
  return (p.gender ?? p.performer?.gender) === GQL.GenderEnum.Male;
}

// Non-male performers first, keeping the recorded order within each group.
function visiblePerformers(
  performers: TombstonePerformer[],
  hideMale: boolean
) {
  const nonMale = performers.filter((p) => !isMale(p));
  if (hideMale) return nonMale;
  return nonMale.concat(performers.filter(isMale));
}

// Links to the library performer if it still exists, else to its stash-box
// page, else plain text.
const TombstonePerformerLink: React.FC<{ performer: TombstonePerformer }> = ({
  performer,
}) => {
  if (performer.performer) {
    return (
      <Link to={`/performers/${performer.performer.id}`}>{performer.name}</Link>
    );
  }

  for (const id of performer.stash_ids) {
    const base = getStashboxBase(id.endpoint);
    if (base) {
      return (
        <ExternalLink href={`${base}performers/${id.stash_id}`}>
          {performer.name}
        </ExternalLink>
      );
    }
  }

  return <>{performer.name}</>;
};

const TombstonePerformers: React.FC<{
  performers: TombstonePerformer[];
  hideMale: boolean;
}> = ({ performers, hideMale }) => {
  const visible = visiblePerformers(performers, hideMale);
  if (visible.length === 0) {
    return (
      <span className="text-muted">
        <FormattedMessage id="none" />
      </span>
    );
  }

  return (
    <div className="tombstone-performers">
      {visible.map((p, i) => (
        <span key={i}>
          <TombstonePerformerLink performer={p} />
        </span>
      ))}
    </div>
  );
};

const TAGS_SHOWN = 8;

const TombstoneTags: React.FC<{ tags: TombstoneData["tag_items"] }> = ({
  tags,
}) => {
  const [open, setOpen] = useState(false);
  if (tags.length === 0) {
    return (
      <span className="text-muted">
        <FormattedMessage id="none" />
      </span>
    );
  }

  return (
    <div className="tombstone-tags">
      {(open ? tags : tags.slice(0, TAGS_SHOWN)).map((t) => (
        <Badge
          key={t.name}
          variant="secondary"
          className={t.tag ? "tag-item tag-link" : "tag-item"}
        >
          {t.tag ? <Link to={`/tags/${t.tag.id}`}>{t.name}</Link> : t.name}
        </Badge>
      ))}
      {tags.length > TAGS_SHOWN && (
        <MoreButton
          hidden={tags.length - TAGS_SHOWN}
          open={open}
          onToggle={() => setOpen(!open)}
        />
      )}
    </div>
  );
};

const CopyButton: React.FC<{
  text: string;
  messageID: string;
  values?: Record<string, string>;
}> = ({ text, messageID, values }) => {
  const intl = useIntl();
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => () => clearTimeout(timer.current), []);

  function onCopy() {
    copyToClipboard(text);
    setCopied(true);
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setCopied(false), 1500);
  }

  const label = intl.formatMessage(
    { id: copied ? "tombstones.copied" : messageID },
    values
  );

  return (
    <Button
      size="sm"
      className={cx("minimal", { "text-success": copied })}
      title={label}
      aria-label={label}
      onClick={onCopy}
    >
      <Icon icon={copied ? faCheck : faCopy} />
    </Button>
  );
};

export const PhashLink: React.FC<{ phash?: string | null }> = ({ phash }) => {
  const intl = useIntl();
  if (!phash) return <span className="text-muted">—</span>;

  return (
    <Link
      to={NavUtils.makeScenesPHashMatchUrl(phash)}
      title={intl.formatMessage({ id: "tombstones.find_phash" })}
    >
      {phash}
    </Link>
  );
};

// The "+N more" / "Show less" toggle of a capped list.
const MoreButton: React.FC<{
  hidden: number;
  open: boolean;
  onToggle: () => void;
}> = ({ hidden, open, onToggle }) => (
  <Button
    variant="outline-primary"
    size="sm"
    className="tombstone-more"
    aria-expanded={open}
    onClick={onToggle}
  >
    {open ? (
      <FormattedMessage id="tombstones.show_less" />
    ) : (
      <FormattedMessage id="tombstones.show_more" values={{ count: hidden }} />
    )}
  </Button>
);

// Long text that starts collapsed to its first line.
const Disclosure: React.FC<{ text: string }> = ({ text }) => {
  const [open, setOpen] = useState(false);
  const firstLine = text.trim().split("\n")[0];

  return (
    <>
      <Button
        variant="link"
        className={cx("minimal tombstone-disclosure", { open })}
        aria-expanded={open}
        onClick={() => setOpen(!open)}
      >
        <Icon icon={open ? faChevronDown : faChevronRight} fixedWidth />
        <span>{open ? <FormattedMessage id="actions.hide" /> : firstLine}</span>
      </Button>
      {open && <p className="pre tombstone-longtext">{text}</p>}
    </>
  );
};

const TombstoneFileDetails: React.FC<{ file: TombstoneFile }> = ({ file }) => {
  const intl = useIntl();

  const info: React.ReactNode[] = [];
  if (file.width && file.height) info.push(`${file.width}×${file.height}`);
  if (file.video_codec) info.push(file.video_codec);
  if (file.bit_rate) {
    info.push(
      intl.formatMessage(
        { id: "megabits_per_second" },
        {
          value: intl.formatNumber(file.bit_rate / 1000000, {
            maximumFractionDigits: 1,
          }),
        }
      )
    );
  }

  return (
    <div className="tombstone-file">
      <div className="tombstone-copy-line tombstone-filename-full">
        <span className="tombstone-mono">{file.basename}</span>
        <CopyButton text={file.basename} messageID="tombstones.copy_filename" />
      </div>
      <div className="tombstone-copy-line text-muted">
        <span className="tombstone-mono">{file.path}</span>
        <CopyButton text={file.path} messageID="tombstones.copy_path" />
      </div>
      <div className="tombstone-mono text-muted">
        <TombstoneFileSize size={file.size} showGiB />
        {info.map((v, i) => (
          <React.Fragment key={i}> · {v}</React.Fragment>
        ))}
      </div>
      <div className="tombstone-mono tombstone-hashes">
        <span className="tombstone-copy-line">
          <span>
            <span className="tombstone-key">phash</span>
            <PhashLink phash={file.phash} />
          </span>
          {file.phash && (
            <CopyButton text={file.phash} messageID="tombstones.copy_phash" />
          )}
        </span>
        {file.oshash && (
          <span>
            <span className="tombstone-key">oshash</span>
            {file.oshash}
          </span>
        )}
        {file.md5 && (
          <span>
            <span className="tombstone-key">md5</span>
            {file.md5}
          </span>
        )}
      </div>
    </div>
  );
};

const TombstoneStashID: React.FC<{
  stashID: TombstoneData["stash_ids"][number];
}> = ({ stashID }) => {
  const { configuration } = useConfigurationContext();
  const { endpoint, stash_id } = stashID;
  const name =
    configuration?.general.stashBoxes.find((sb) => sb.endpoint === endpoint)
      ?.name || endpoint;
  const base = getStashboxBase(endpoint);

  return (
    <>
      <dt className="tombstone-endpoint">{name}</dt>
      <dd className="tombstone-copy-line">
        <span className="tombstone-mono">
          {base ? (
            <ExternalLink href={`${base}scenes/${stash_id}`}>
              {stash_id}
            </ExternalLink>
          ) : (
            stash_id
          )}
        </span>
        <CopyButton
          text={stash_id}
          messageID="tombstones.copy_stash_id"
          values={{ endpoint: name }}
        />
      </dd>
    </>
  );
};

const URLS_SHOWN = 2;

const TombstoneURLs: React.FC<{ urls: string[] }> = ({ urls }) => {
  const [open, setOpen] = useState(false);
  if (urls.length === 0) {
    return (
      <span className="text-muted">
        <FormattedMessage id="none" />
      </span>
    );
  }

  const shown = open ? urls : urls.slice(0, URLS_SHOWN);

  return (
    <div className="tombstone-urls">
      {shown.map((u, i) => (
        <ExternalLink key={i} href={u} title={u}>
          {u.replace(/^https?:\/\//, "").replace(/^www\./, "")}
        </ExternalLink>
      ))}
      {urls.length > URLS_SHOWN && (
        <MoreButton
          hidden={urls.length - URLS_SHOWN}
          open={open}
          onToggle={() => setOpen(!open)}
        />
      )}
    </div>
  );
};

// Rating, O-counter and play count of the scene when it was deleted.
const TombstoneStats: React.FC<{ tombstone: TombstoneData }> = ({
  tombstone,
}) => {
  const intl = useIntl();
  const { configuration } = useConfigurationContext();
  const options =
    configuration?.ui.ratingSystemOptions ?? defaultRatingSystemOptions;
  const rating = convertToRatingFormat(tombstone.rating100, options);
  const max = options.type === RatingSystemType.Decimal ? 10 : 5;

  const parts = [
    rating === null
      ? intl.formatMessage({ id: "tombstones.unrated" })
      : intl.formatMessage(
          { id: "tombstones.rating" },
          { value: intl.formatNumber(rating), max }
        ),
    intl.formatMessage(
      { id: "tombstones.o_count" },
      { count: tombstone.o_counter ?? 0 }
    ),
    intl.formatMessage(
      { id: "tombstones.play_count" },
      { count: tombstone.play_count ?? 0 }
    ),
  ];

  return <span className="text-muted">{parts.join(" · ")}</span>;
};

// The expanded part of a table row: everything the row has no column for.
// Files on the left, identifiers on the right, scene metadata below.
export const TombstoneDetails: React.FC<{
  tombstone: TombstoneData;
  matchedKeys: GQL.TombstoneMatchKey[];
  hideMale: boolean;
}> = ({ tombstone, matchedKeys, hideMale }) => {
  const intl = useIntl();

  return (
    <div role="row" className="tombstone-details">
      <div role="cell">
        <div className="tombstone-files">
          {tombstone.files.length === 0 && (
            <span className="text-muted">
              <FormattedMessage id="tombstones.no_files" />
            </span>
          )}
          {tombstone.files.map((f) => (
            <TombstoneFileDetails key={f.id} file={f} />
          ))}
        </div>
        <dl>
          <dt>
            <FormattedMessage id="scene_code" />
          </dt>
          <dd className="tombstone-mono">{tombstone.code || "—"}</dd>
          {tombstone.stash_ids.map((s) => (
            <TombstoneStashID key={`${s.endpoint}/${s.stash_id}`} stashID={s} />
          ))}
          {tombstone.stash_ids.length === 0 && (
            <>
              <dt>
                <FormattedMessage id="stash_ids" />
              </dt>
              <dd className="text-muted">
                <FormattedMessage id="none" />
              </dd>
            </>
          )}
          <dt>
            <FormattedMessage id="urls" />
          </dt>
          <dd>
            <TombstoneURLs urls={tombstone.urls} />
          </dd>
        </dl>
        <dl className="tombstone-scene">
          <dt>
            <FormattedMessage id="performers" />
          </dt>
          <dd>
            <TombstonePerformers
              performers={tombstone.performers}
              hideMale={hideMale}
            />
          </dd>
          <dt>
            <FormattedMessage id="tombstones.created" />
          </dt>
          <dd>
            {tombstone.scene_created_at && (
              <span className="tombstone-created">
                {TextUtils.formatDateTime(intl, tombstone.scene_created_at)}
              </span>
            )}
            <TombstoneStats tombstone={tombstone} />
          </dd>
          <dt>
            <FormattedMessage
              id="tombstones.tags_count"
              values={{ count: tombstone.tag_items.length }}
            />
          </dt>
          <dd>
            <TombstoneTags tags={tombstone.tag_items} />
          </dd>
          {tombstone.details && (
            <>
              <dt>
                <FormattedMessage id="details" />
              </dt>
              <dd>
                <Disclosure text={tombstone.details} />
              </dd>
            </>
          )}
          {tombstone.original_title && (
            <>
              <dt>
                <FormattedMessage id="tombstones.original_title" />
              </dt>
              <dd>
                <Disclosure text={tombstone.original_title} />
              </dd>
            </>
          )}
          {tombstone.original_details && (
            <>
              <dt>
                <FormattedMessage id="tombstones.original_details" />
              </dt>
              <dd>
                <Disclosure text={tombstone.original_details} />
              </dd>
            </>
          )}
          {matchedKeys.length > 0 && (
            <>
              <dt>
                <FormattedMessage id="tombstones.matched_on" />
              </dt>
              <dd>
                {matchedKeys.map((k) => (
                  <Badge key={k} variant="info" className="matched-key">
                    <FormattedMessage id={matchKeyMessageIDs[k]} />
                  </Badge>
                ))}
              </dd>
            </>
          )}
        </dl>
      </div>
    </div>
  );
};
