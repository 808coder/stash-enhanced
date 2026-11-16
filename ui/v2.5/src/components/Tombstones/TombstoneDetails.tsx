import React, { useEffect, useRef, useState } from "react";
import { Badge, Button } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { Link } from "react-router-dom";
import cx from "classnames";
import { faCheck, faCopy } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { copyToClipboard } from "src/utils/clipboard";
import NavUtils from "src/utils/navigation";
import { getStashboxBase } from "src/utils/stashbox";
import TextUtils from "src/utils/text";
import { ExternalLink } from "../Shared/ExternalLink";
import { FileSize } from "../Shared/FileSize";
import { Icon } from "../Shared/Icon";
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

const TombstoneTags: React.FC<{ tags: TombstoneData["tag_items"] }> = ({
  tags,
}) => {
  if (tags.length === 0) {
    return (
      <span className="text-muted">
        <FormattedMessage id="none" />
      </span>
    );
  }

  return (
    <div className="tombstone-tags">
      {tags.map((t) => (
        <Badge
          key={t.name}
          variant="secondary"
          className={t.tag ? "tag-item tag-link" : "tag-item"}
        >
          {t.tag ? <Link to={`/tags/${t.tag.id}`}>{t.name}</Link> : t.name}
        </Badge>
      ))}
    </div>
  );
};

const CopyButton: React.FC<{ text: string; messageID: string }> = ({
  text,
  messageID,
}) => {
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

  const label = intl.formatMessage({
    id: copied ? "tombstones.copied" : messageID,
  });

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

const TombstoneFileDetails: React.FC<{ file: TombstoneFile }> = ({ file }) => (
  <dl className="tombstone-file">
    <dt>
      <FormattedMessage id="filename" />
    </dt>
    <dd className="tombstone-mono tombstone-copy-line">
      <span>{file.basename}</span>
      <CopyButton text={file.basename} messageID="tombstones.copy_filename" />
    </dd>
    <dt>
      <FormattedMessage id="path" />
    </dt>
    <dd className="tombstone-mono tombstone-copy-line">
      <span>{file.path}</span>
      <CopyButton text={file.path} messageID="tombstones.copy_path" />
    </dd>
    <dt>
      <FormattedMessage id="tombstones.file" />
    </dt>
    <dd className="tombstone-mono">
      <FileSize size={file.size} />
      {file.width && file.height ? ` · ${file.width}×${file.height}` : ""}
      {file.video_codec ? ` · ${file.video_codec}` : ""}
    </dd>
    {file.oshash && (
      <>
        <dt>oshash</dt>
        <dd className="tombstone-mono">{file.oshash}</dd>
      </>
    )}
    {file.md5 && (
      <>
        <dt>md5</dt>
        <dd className="tombstone-mono">{file.md5}</dd>
      </>
    )}
    <dt>phash</dt>
    <dd className="tombstone-mono">
      <PhashLink phash={file.phash} />
    </dd>
  </dl>
);

// The expanded part of a table row: everything the row has no column for.
export const TombstoneDetails: React.FC<{
  tombstone: TombstoneData;
  matchedKeys: GQL.TombstoneMatchKey[];
  hideMale: boolean;
}> = ({ tombstone, matchedKeys, hideMale }) => {
  const intl = useIntl();

  return (
    <div role="row" className="tombstone-details">
      <div role="cell">
        <dl>
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
            <FormattedMessage id="tags" />
          </dt>
          <dd>
            <TombstoneTags tags={tombstone.tag_items} />
          </dd>
          <dt>
            <FormattedMessage id="tombstones.deleted" />
          </dt>
          <dd>
            {TextUtils.formatDateTime(intl, tombstone.deleted_at)}
            {" · "}
            <FormattedMessage
              id={
                tombstone.origin === GQL.TombstoneOrigin.Manual
                  ? "tombstones.origin_manual"
                  : "tombstones.origin_clean"
              }
            />
          </dd>
          <dt>
            <FormattedMessage id="tombstones.reason" />
          </dt>
          <dd className="tombstone-reason">
            {tombstone.reason || <FormattedMessage id="tombstones.no_reason" />}
          </dd>
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
        <div>
          {tombstone.files.length === 0 && (
            <span className="text-muted">
              <FormattedMessage id="tombstones.no_files" />
            </span>
          )}
          {tombstone.files.map((f) => (
            <TombstoneFileDetails key={f.id} file={f} />
          ))}
        </div>
      </div>
    </div>
  );
};
