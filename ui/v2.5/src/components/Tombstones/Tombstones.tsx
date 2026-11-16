import React, { useEffect, useRef, useState } from "react";
import { Badge, Button, ButtonGroup, Card, Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { Link } from "react-router-dom";
import cx from "classnames";
import {
  faArrowDown,
  faArrowUp,
  faBolt,
  faCheck,
  faChevronRight,
  faCopy,
  faPencilAlt,
  faTrash,
  faUser,
} from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { useDebouncedState } from "src/hooks/debounce";
import TextUtils from "src/utils/text";
import NavUtils from "src/utils/navigation";
import { getStashboxBase } from "src/utils/stashbox";
import { ClearableInput } from "../Shared/ClearableInput";
import { ErrorMessage } from "../Shared/ErrorMessage";
import { ExternalLink } from "../Shared/ExternalLink";
import { FileSize } from "../Shared/FileSize";
import { Icon } from "../Shared/Icon";
import { LoadingIndicator } from "../Shared/LoadingIndicator";
import { ModalComponent } from "../Shared/Modal";
import { ReasonSelect } from "./ReasonSelect";
import { Pagination } from "../List/Pagination";

const phashRE = /^[0-9a-fA-F]{16}$/;

// One search term is tried against every filter key; keys are OR-combined.
function getFilter(term: string): GQL.TombstoneFilterInput | undefined {
  const t = term.trim();
  if (!t) return undefined;

  return {
    url: t,
    stash_id: t,
    oshash: t,
    md5: t,
    code: t,
    title: t,
    basename: t,
    phash: phashRE.test(t) ? { value: t, distance: 0 } : undefined,
  };
}

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

type TombstoneData = GQL.TombstoneDataFragment;
type TombstonePerformer = TombstoneData["performers"][number];

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

const EditReasonDialog: React.FC<{
  tombstone: TombstoneData;
  onClose: (saved: boolean) => void;
}> = ({ tombstone, onClose }) => {
  const intl = useIntl();
  const Toast = useToast();
  const [reason, setReason] = useState(tombstone.reason);
  const [saving, setSaving] = useState(false);
  const [update] = GQL.useTombstoneUpdateMutation();

  async function onSave() {
    setSaving(true);
    try {
      await update({
        variables: { input: { id: tombstone.id, reason: reason.trim() } },
      });
      Toast.success(intl.formatMessage({ id: "tombstones.toast.updated" }));
      onClose(true);
    } catch (e) {
      Toast.error(e);
      setSaving(false);
    }
  }

  return (
    <ModalComponent
      show
      icon={faPencilAlt}
      header={intl.formatMessage({ id: "tombstones.edit_reason" })}
      accept={{
        onClick: onSave,
        text: intl.formatMessage({ id: "actions.save" }),
      }}
      cancel={{
        onClick: () => onClose(false),
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={saving}
    >
      <ReasonSelect value={reason} onChange={setReason} />
    </ModalComponent>
  );
};

const DeleteTombstoneDialog: React.FC<{
  tombstone: TombstoneData;
  onClose: (deleted: boolean) => void;
}> = ({ tombstone, onClose }) => {
  const intl = useIntl();
  const Toast = useToast();
  const [deleting, setDeleting] = useState(false);
  const [destroy] = GQL.useTombstoneDestroyMutation();

  async function onDelete() {
    setDeleting(true);
    try {
      await destroy({ variables: { id: tombstone.id } });
      Toast.success(intl.formatMessage({ id: "tombstones.toast.deleted" }));
      onClose(true);
    } catch (e) {
      Toast.error(e);
      setDeleting(false);
    }
  }

  return (
    <ModalComponent
      show
      icon={faTrash}
      header={intl.formatMessage({ id: "tombstones.delete_title" })}
      accept={{
        variant: "danger",
        onClick: onDelete,
        text: intl.formatMessage({ id: "actions.delete" }),
      }}
      cancel={{
        onClick: () => onClose(false),
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={deleting}
    >
      <p>
        <FormattedMessage
          id="tombstones.delete_desc"
          values={{ title: tombstone.title || tombstone.original_title }}
        />
      </p>
    </ModalComponent>
  );
};

type SortKey =
  | "title"
  | "deleted_at"
  | "origin"
  | "reason"
  | "size"
  | "resolution"
  | "basename"
  | "phash";

interface ISort {
  key: SortKey;
  direction: GQL.SortDirectionEnum;
}

const defaultSort: ISort = {
  key: "deleted_at",
  direction: GQL.SortDirectionEnum.Desc,
};

// Columns that start descending when first sorted by.
const descendingFirst: SortKey[] = ["deleted_at", "size", "resolution"];

const SortHeader: React.FC<{
  sortKey: SortKey;
  messageID: string;
  sort: ISort;
  onSort: (key: SortKey) => void;
  className?: string;
}> = ({ sortKey, messageID, sort, onSort, className }) => {
  const active = sort.key === sortKey;
  const ascending = sort.direction === GQL.SortDirectionEnum.Asc;

  let ariaSort: "ascending" | "descending" | "none" = "none";
  if (active) ariaSort = ascending ? "ascending" : "descending";

  return (
    <div role="columnheader" aria-sort={ariaSort} className={className}>
      <button
        type="button"
        className={cx("tombstone-sort", { active })}
        onClick={() => onSort(sortKey)}
      >
        <FormattedMessage id={messageID} />
        <Icon icon={active && ascending ? faArrowUp : faArrowDown} />
      </button>
    </div>
  );
};

// Reasons are free text from a configurable list, so the colour is a best
// effort on the default reasons and neutral for everything else.
function reasonVariant(reason: string) {
  if (/duplicate/i.test(reason)) return "duplicate";
  if (/replaced/i.test(reason)) return "replaced";
  if (/quality|broken/i.test(reason)) return "quality";
  return "neutral";
}

// navigator.clipboard only exists in secure contexts (https or localhost);
// Stash is often served over plain http on a LAN, hence the fallback.
function copyText(text: string) {
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

const CopyButton: React.FC<{ text: string; messageID: string }> = ({
  text,
  messageID,
}) => {
  const intl = useIntl();
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => () => clearTimeout(timer.current), []);

  function onCopy() {
    copyText(text);
    setCopied(true);
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setCopied(false), 1500);
  }

  const label = intl.formatMessage({
    id: copied ? "tombstones.copied" : messageID,
  });

  return (
    <button
      type="button"
      className={cx("tombstone-copy", { copied })}
      title={label}
      aria-label={label}
      onClick={onCopy}
    >
      <Icon icon={copied ? faCheck : faCopy} />
    </button>
  );
};

const PhashLink: React.FC<{ phash?: string | null }> = ({ phash }) => {
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

const TombstoneDetails: React.FC<{
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
            <dl key={f.id} className="tombstone-file">
              <dt>
                <FormattedMessage id="filename" />
              </dt>
              <dd className="tombstone-mono tombstone-copy-line">
                <span>{f.basename}</span>
                <CopyButton
                  text={f.basename}
                  messageID="tombstones.copy_filename"
                />
              </dd>
              <dt>
                <FormattedMessage id="path" />
              </dt>
              <dd className="tombstone-mono tombstone-copy-line">
                <span>{f.path}</span>
                <CopyButton text={f.path} messageID="tombstones.copy_path" />
              </dd>
              <dt>
                <FormattedMessage id="tombstones.file" />
              </dt>
              <dd className="tombstone-mono">
                <FileSize size={f.size} />
                {f.width && f.height ? ` · ${f.width}×${f.height}` : ""}
                {f.video_codec ? ` · ${f.video_codec}` : ""}
              </dd>
              {f.oshash && (
                <>
                  <dt>oshash</dt>
                  <dd className="tombstone-mono">{f.oshash}</dd>
                </>
              )}
              {f.md5 && (
                <>
                  <dt>md5</dt>
                  <dd className="tombstone-mono">{f.md5}</dd>
                </>
              )}
              <dt>phash</dt>
              <dd className="tombstone-mono">
                <PhashLink phash={f.phash} />
              </dd>
            </dl>
          ))}
        </div>
      </div>
    </div>
  );
};

const TombstoneRow: React.FC<{
  tombstone: TombstoneData;
  matchedKeys: GQL.TombstoneMatchKey[];
  hideMale: boolean;
  open: boolean;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
}> = ({
  tombstone,
  matchedKeys,
  hideMale,
  open,
  onToggle,
  onEdit,
  onDelete,
}) => {
  const intl = useIntl();
  const title = tombstone.title || tombstone.original_title;
  const meta = [tombstone.studio?.name, tombstone.date]
    .filter((v) => !!v)
    .join(" · ");
  const manual = tombstone.origin === GQL.TombstoneOrigin.Manual;
  // the first file is the one the file columns sort by
  const file = tombstone.files.length > 0 ? tombstone.files[0] : undefined;
  const moreFiles = tombstone.files.length - 1;

  return (
    <div role="rowgroup">
      <div role="row" className={cx("tombstone-grid tombstone-row", { open })}>
        <div role="cell">
          <button
            type="button"
            className={cx("tombstone-toggle", { open })}
            aria-expanded={open}
            aria-label={intl.formatMessage({ id: "tombstones.show_details" })}
            onClick={onToggle}
          >
            <Icon icon={faChevronRight} />
          </button>
        </div>
        <div role="cell" className="tombstone-title-cell">
          <button
            type="button"
            className={cx("tombstone-title", { untitled: !title })}
            title={title || file?.basename}
            onClick={onToggle}
          >
            {title || file?.basename || (
              <FormattedMessage id="tombstones.untitled" />
            )}
          </button>
          {meta && <div className="tombstone-sub text-muted">{meta}</div>}
        </div>
        <div role="cell">
          <div>{TextUtils.formatDate(intl, tombstone.deleted_at, false)}</div>
          <div className="tombstone-sub text-muted">
            {intl.formatTime(tombstone.deleted_at)}
          </div>
        </div>
        <div role="cell">
          <span
            className={cx("tombstone-badge", manual ? "manual" : "clean")}
            title={intl.formatMessage({
              id: manual
                ? "tombstones.origin_manual"
                : "tombstones.origin_clean",
            })}
          >
            <Icon icon={manual ? faUser : faBolt} />
            <FormattedMessage
              id={
                manual
                  ? "tombstones.origin_manual_short"
                  : "tombstones.origin_clean_short"
              }
            />
          </span>
        </div>
        <div role="cell">
          {tombstone.reason ? (
            <span
              className={cx("tombstone-badge", reasonVariant(tombstone.reason))}
              title={tombstone.reason}
            >
              {tombstone.reason}
            </span>
          ) : (
            <span className="text-muted">
              <FormattedMessage id="tombstones.no_reason_short" />
            </span>
          )}
        </div>
        <div role="cell" className="tombstone-mono text-right">
          {file ? <FileSize size={file.size} /> : "—"}
        </div>
        <div
          role="cell"
          className="tombstone-mono text-right"
          title={file?.height ? `${file.width}×${file.height}` : undefined}
        >
          {file?.height ? `${file.height}p` : "—"}
        </div>
        <div role="cell" className="tombstone-mono tombstone-filename">
          <span title={file?.basename}>{file?.basename ?? "—"}</span>
          {moreFiles > 0 && (
            <span
              className="text-muted"
              title={intl.formatMessage(
                { id: "tombstones.more_files" },
                { count: moreFiles }
              )}
            >
              +{moreFiles}
            </span>
          )}
        </div>
        <div role="cell" className="tombstone-mono">
          <PhashLink phash={file?.phash} />
        </div>
        <div role="cell" className="tombstone-actions">
          <Button
            variant="secondary"
            title={intl.formatMessage({ id: "tombstones.edit_reason" })}
            aria-label={intl.formatMessage({ id: "tombstones.edit_reason" })}
            onClick={onEdit}
          >
            <Icon icon={faPencilAlt} />
          </Button>
          <Button
            variant="danger"
            title={intl.formatMessage({ id: "tombstones.delete_title" })}
            aria-label={intl.formatMessage({ id: "tombstones.delete_title" })}
            onClick={onDelete}
          >
            <Icon icon={faTrash} />
          </Button>
        </div>
      </div>
      {open && (
        <TombstoneDetails
          tombstone={tombstone}
          matchedKeys={matchedKeys}
          hideMale={hideMale}
        />
      )}
    </div>
  );
};

const pageSizes = [10, 25, 50, 100];

const originFilters: {
  origin?: GQL.TombstoneOrigin;
  messageID: string;
}[] = [
  { messageID: "tombstones.origin_all" },
  {
    origin: GQL.TombstoneOrigin.Manual,
    messageID: "tombstones.origin_manual_short",
  },
  {
    origin: GQL.TombstoneOrigin.Clean,
    messageID: "tombstones.origin_clean_short",
  },
];

export const Tombstones: React.FC = () => {
  const intl = useIntl();

  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [origin, setOrigin] = useState<GQL.TombstoneOrigin>();
  const [sort, setSort] = useState(defaultSort);
  const [searchInput, onSearchChange] = useDebouncedState<string>(
    "",
    (v) => {
      setSearch(v);
      setPage(1);
    },
    300
  );
  const [opened, setOpened] = useState<Record<string, boolean>>({});
  const [editing, setEditing] = useState<TombstoneData>();
  const [deleting, setDeleting] = useState<TombstoneData>();
  // not persisted, like the detail panel toggle (#37)
  const [hideMale, setHideMale] = useState(false);

  const query = GQL.useFindTombstonesQuery({
    fetchPolicy: "no-cache",
    variables: {
      tombstone_filter: { ...getFilter(search), origin },
      filter: {
        page,
        per_page: pageSize,
        sort: sort.key,
        direction: sort.direction,
      },
    },
  });
  const { loading, error, refetch } = query;
  // keep showing the last result while the next one loads, so that sorting
  // or paging does not unmount the table
  const data = query.data ?? query.previousData;

  function onSort(key: SortKey) {
    const { Asc, Desc } = GQL.SortDirectionEnum;
    if (sort.key === key) {
      setSort({ key, direction: sort.direction === Asc ? Desc : Asc });
    } else {
      setSort({ key, direction: descendingFirst.includes(key) ? Desc : Asc });
    }
    setPage(1);
  }

  function onDeleted() {
    // step back when the last tombstone of the last page is gone
    if (data && data.findTombstones.matches.length === 1 && page > 1) {
      setPage(page - 1);
    } else {
      refetch();
    }
  }

  function renderOriginFilter() {
    const result = data?.findTombstones;
    const counts = result && {
      all: result.manual_count + result.clean_count,
      [GQL.TombstoneOrigin.Manual]: result.manual_count,
      [GQL.TombstoneOrigin.Clean]: result.clean_count,
    };

    return (
      <ButtonGroup
        role="group"
        aria-label={intl.formatMessage({ id: "tombstones.origin" })}
      >
        {originFilters.map((f) => (
          <Button
            key={f.origin ?? "all"}
            variant="secondary"
            active={origin === f.origin}
            aria-pressed={origin === f.origin}
            onClick={() => {
              setOrigin(f.origin);
              setPage(1);
            }}
          >
            <FormattedMessage id={f.messageID} />
            {counts && (
              <span className="tombstone-count">
                {" · "}
                {intl.formatNumber(counts[f.origin ?? "all"])}
              </span>
            )}
          </Button>
        ))}
      </ButtonGroup>
    );
  }

  function renderTable() {
    if (error) return <ErrorMessage error={error.message} />;
    if (!data) return <LoadingIndicator />;

    const { count, matches } = data.findTombstones;
    const first = Math.min((page - 1) * pageSize + 1, count);
    const last = Math.min(first + matches.length - 1, count);

    const header = (key: SortKey, messageID: string, className?: string) => (
      <SortHeader
        sortKey={key}
        messageID={messageID}
        sort={sort}
        onSort={onSort}
        className={className}
      />
    );

    return (
      <>
        <div className={cx("tombstone-table-frame", { loading })}>
          <div className="tombstone-table-scroll">
            <div
              role="table"
              className="tombstone-table"
              aria-label={intl.formatMessage({ id: "tombstones.title" })}
              aria-rowcount={count}
              aria-busy={loading}
            >
              <div role="rowgroup">
                <div role="row" className="tombstone-grid tombstone-head">
                  <div role="columnheader">
                    <span className="sr-only">
                      <FormattedMessage id="details" />
                    </span>
                  </div>
                  {header("title", "title")}
                  {header("deleted_at", "tombstones.deleted")}
                  {header("origin", "tombstones.origin")}
                  {header("reason", "tombstones.reason")}
                  {header("size", "tombstones.size", "text-right")}
                  {header(
                    "resolution",
                    "tombstones.resolution_short",
                    "text-right"
                  )}
                  {header("basename", "filename")}
                  {header("phash", "tombstones.match_keys.phash")}
                  <div role="columnheader">
                    <span className="sr-only">
                      <FormattedMessage id="tombstones.actions" />
                    </span>
                  </div>
                </div>
              </div>
              {matches.map((m) => (
                <TombstoneRow
                  key={m.tombstone.id}
                  tombstone={m.tombstone}
                  matchedKeys={m.matched_keys}
                  hideMale={hideMale}
                  open={!!opened[m.tombstone.id]}
                  onToggle={() =>
                    setOpened({
                      ...opened,
                      [m.tombstone.id]: !opened[m.tombstone.id],
                    })
                  }
                  onEdit={() => setEditing(m.tombstone)}
                  onDelete={() => setDeleting(m.tombstone)}
                />
              ))}
              {matches.length === 0 && (
                <div role="rowgroup">
                  <div role="row" className="tombstone-empty">
                    <div role="cell" className="text-muted">
                      <FormattedMessage id="tombstones.empty" />
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="tombstone-footer">
          <div className="tombstone-footer-info">
            <span>
              <strong>
                {count === 0
                  ? intl.formatNumber(0)
                  : `${intl.formatNumber(first)}–${intl.formatNumber(last)}`}
              </strong>{" "}
              <span className="text-muted">
                <FormattedMessage id="tombstones.of_total" values={{ count }} />
              </span>
            </span>
            <Form.Group
              controlId="tombstones-page-size"
              className="tombstone-page-size text-muted"
            >
              <Form.Label>
                <FormattedMessage id="tombstones.per_page" />
              </Form.Label>
              <Form.Control
                as="select"
                className="input-control"
                value={pageSize}
                onChange={(e) => {
                  setPageSize(Number(e.currentTarget.value));
                  setPage(1);
                }}
              >
                {pageSizes.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </Form.Control>
            </Form.Group>
          </div>
          <Pagination
            itemsPerPage={pageSize}
            currentPage={page}
            totalItems={count}
            metadataByline={[]}
            onChangePage={setPage}
          />
        </div>
      </>
    );
  }

  return (
    <Card id="tombstones" className="col mx-auto">
      <div className="tombstone-heading">
        <h4>
          <FormattedMessage id="tombstones.title" />
        </h4>
        <p className="text-muted">
          <FormattedMessage id="tombstones.description" />
        </p>
      </div>
      <div className="tombstone-toolbar">
        <ClearableInput
          className="tombstone-search"
          value={searchInput}
          setValue={onSearchChange}
          placeholder={intl.formatMessage({ id: "tombstones.search" })}
        />
        {renderOriginFilter()}
        <Form.Check
          type="switch"
          id="tombstones-hide-male"
          className="tombstone-hide-male"
          checked={hideMale}
          onChange={() => setHideMale(!hideMale)}
          label={<FormattedMessage id="hide_male_performers" />}
        />
      </div>
      {renderTable()}
      {editing && (
        <EditReasonDialog
          tombstone={editing}
          onClose={(saved) => {
            setEditing(undefined);
            if (saved) refetch();
          }}
        />
      )}
      {deleting && (
        <DeleteTombstoneDialog
          tombstone={deleting}
          onClose={(deleted) => {
            setDeleting(undefined);
            if (deleted) onDeleted();
          }}
        />
      )}
    </Card>
  );
};

export default Tombstones;
