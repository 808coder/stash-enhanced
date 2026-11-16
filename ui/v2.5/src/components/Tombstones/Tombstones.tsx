import React, { useState } from "react";
import { Badge, Button, ButtonGroup, Card, Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { Link } from "react-router-dom";
import { faPencilAlt, faTrash } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { useDebouncedState } from "src/hooks/debounce";
import TextUtils from "src/utils/text";
import { getStashboxBase } from "src/utils/stashbox";
import { ClearableInput } from "../Shared/ClearableInput";
import { ErrorMessage } from "../Shared/ErrorMessage";
import { ExternalLink } from "../Shared/ExternalLink";
import { FileSize } from "../Shared/FileSize";
import { Icon } from "../Shared/Icon";
import { LoadingIndicator } from "../Shared/LoadingIndicator";
import { ModalComponent } from "../Shared/Modal";
import { Pagination } from "../List/Pagination";

const PAGE_SIZE = 25;

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
  if (visible.length === 0) return null;

  return (
    <div className="mt-2 tombstone-performers">
      <strong>
        <FormattedMessage id="performers" />:{" "}
      </strong>
      {visible.map((p, i) => (
        <React.Fragment key={i}>
          {i > 0 && ", "}
          <TombstonePerformerLink performer={p} />
        </React.Fragment>
      ))}
    </div>
  );
};

const TombstoneTags: React.FC<{ tags: TombstoneData["tag_items"] }> = ({
  tags,
}) => {
  if (tags.length === 0) return null;

  return (
    <div className="mt-2 tombstone-tags">
      <strong>
        <FormattedMessage id="tags" />:{" "}
      </strong>
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
      <Form.Control
        as="textarea"
        rows={4}
        className="text-input"
        value={reason}
        placeholder={intl.formatMessage({ id: "tombstones.reason" })}
        onChange={(e) => setReason(e.currentTarget.value)}
      />
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

const TombstoneCard: React.FC<{
  tombstone: TombstoneData;
  matchedKeys: GQL.TombstoneMatchKey[];
  hideMale: boolean;
  onEdit: () => void;
  onDelete: () => void;
}> = ({ tombstone, matchedKeys, hideMale, onEdit, onDelete }) => {
  const intl = useIntl();
  const title = tombstone.title || tombstone.original_title;

  return (
    <Card className="mb-3 tombstone">
      <Card.Body>
        <div className="d-flex align-items-start">
          <div className="mr-auto">
            <h5>{title || <FormattedMessage id="tombstones.untitled" />}</h5>
            <div className="text-muted">
              <FormattedMessage
                id="tombstones.deleted_at"
                values={{
                  date: TextUtils.formatDateTime(intl, tombstone.deleted_at),
                }}
              />
              {" · "}
              <FormattedMessage
                id={
                  tombstone.origin === GQL.TombstoneOrigin.Manual
                    ? "tombstones.origin_manual"
                    : "tombstones.origin_clean"
                }
              />
              {tombstone.studio ? ` · ${tombstone.studio.name}` : ""}
              {tombstone.date ? ` · ${tombstone.date}` : ""}
            </div>
          </div>
          <ButtonGroup>
            <Button
              variant="secondary"
              title={intl.formatMessage({ id: "tombstones.edit_reason" })}
              onClick={onEdit}
            >
              <Icon icon={faPencilAlt} />
            </Button>
            <Button
              variant="danger"
              title={intl.formatMessage({ id: "actions.delete" })}
              onClick={onDelete}
            >
              <Icon icon={faTrash} />
            </Button>
          </ButtonGroup>
        </div>

        {matchedKeys.length > 0 && (
          <div className="mt-2">
            <span className="mr-2">
              <FormattedMessage id="tombstones.matched_on" />
            </span>
            {matchedKeys.map((k) => (
              <Badge key={k} variant="info" className="matched-key">
                <FormattedMessage id={matchKeyMessageIDs[k]} />
              </Badge>
            ))}
          </div>
        )}

        <TombstonePerformers
          performers={tombstone.performers}
          hideMale={hideMale}
        />
        <TombstoneTags tags={tombstone.tag_items} />

        <div className="mt-2 tombstone-reason">
          <strong>
            <FormattedMessage id="tombstones.reason" />:{" "}
          </strong>
          {tombstone.reason || (
            <span className="text-muted">
              <FormattedMessage id="tombstones.no_reason" />
            </span>
          )}
        </div>

        {tombstone.files.length > 0 && (
          <ul className="mt-2 mb-0 list-unstyled">
            {tombstone.files.map((f) => (
              <li key={f.id} className="tombstone-file-path">
                <div>
                  <strong>
                    <FormattedMessage id="filename" />:
                  </strong>{" "}
                  {f.basename}
                </div>
                <div>
                  <strong>
                    <FormattedMessage id="path" />:
                  </strong>{" "}
                  {f.path}
                </div>
                <div className="text-muted">
                  <FileSize size={f.size} />
                  {f.width && f.height ? ` · ${f.width}x${f.height}` : ""}
                  {f.video_codec ? ` · ${f.video_codec}` : ""}
                  {f.oshash ? ` · oshash ${f.oshash}` : ""}
                  {f.md5 ? ` · md5 ${f.md5}` : ""}
                  {f.phash ? ` · phash ${f.phash}` : ""}
                </div>
              </li>
            ))}
          </ul>
        )}
      </Card.Body>
    </Card>
  );
};

export const Tombstones: React.FC = () => {
  const intl = useIntl();

  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [searchInput, onSearchChange] = useDebouncedState<string>(
    "",
    (v) => {
      setSearch(v);
      setPage(1);
    },
    300
  );
  const [editing, setEditing] = useState<TombstoneData>();
  const [deleting, setDeleting] = useState<TombstoneData>();
  // not persisted, like the detail panel toggle (#37)
  const [hideMale, setHideMale] = useState(false);

  const { data, loading, error, refetch } = GQL.useFindTombstonesQuery({
    fetchPolicy: "no-cache",
    variables: {
      tombstone_filter: getFilter(search),
      filter: { page, per_page: PAGE_SIZE },
    },
  });

  function renderResults() {
    if (error) return <ErrorMessage error={error.message} />;
    if (!data) return <LoadingIndicator />;

    const { count, matches } = data.findTombstones;

    return (
      <>
        <div className="d-flex mt-2 mb-2">
          <h6 className="mr-auto align-self-center">
            <FormattedMessage id="tombstones.found" values={{ count }} />
          </h6>
          <Form.Check
            type="switch"
            id="tombstones-hide-male"
            className="align-self-center mr-3"
            checked={hideMale}
            onChange={() => setHideMale(!hideMale)}
            label={<FormattedMessage id="hide_male_performers" />}
          />
          <Pagination
            itemsPerPage={PAGE_SIZE}
            currentPage={page}
            totalItems={count}
            metadataByline={[]}
            onChangePage={setPage}
          />
        </div>
        {loading && <LoadingIndicator />}
        {matches.length === 0 && !loading && (
          <p className="text-muted">
            <FormattedMessage id="tombstones.empty" />
          </p>
        )}
        {matches.map((m) => (
          <TombstoneCard
            key={m.tombstone.id}
            tombstone={m.tombstone}
            matchedKeys={m.matched_keys}
            hideMale={hideMale}
            onEdit={() => setEditing(m.tombstone)}
            onDelete={() => setDeleting(m.tombstone)}
          />
        ))}
      </>
    );
  }

  return (
    <Card id="tombstones" className="col col-xl-10 mx-auto">
      <div className="px-4 pt-3">
        <h4>
          <FormattedMessage id="tombstones.title" />
        </h4>
        <p className="text-muted">
          <FormattedMessage id="tombstones.description" />
        </p>
        <ClearableInput
          value={searchInput}
          setValue={onSearchChange}
          placeholder={intl.formatMessage({ id: "tombstones.search" })}
        />
      </div>
      <div className="px-4 pb-3">{renderResults()}</div>
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
            if (deleted) refetch();
          }}
        />
      )}
    </Card>
  );
};

export default Tombstones;
