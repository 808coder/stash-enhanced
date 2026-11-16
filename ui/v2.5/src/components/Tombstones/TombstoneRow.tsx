import React from "react";
import { Badge, Button } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import cx from "classnames";
import {
  faBolt,
  faChevronDown,
  faChevronRight,
  faPencilAlt,
  faTrash,
  faUser,
} from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import TextUtils from "src/utils/text";
import { ExpandCollapseButton } from "../Shared/CollapseButton";
import { FileSize } from "../Shared/FileSize";
import { Icon } from "../Shared/Icon";
import { TruncatedText } from "../Shared/TruncatedText";
import { PhashLink, TombstoneDetails } from "./TombstoneDetails";
import { TombstoneData } from "./types";

// Reasons are free text from a configurable list, so the colour is a best
// effort on the default reasons and neutral for everything else.
function reasonVariant(reason: string) {
  if (/duplicate/i.test(reason)) return "info";
  if (/replaced/i.test(reason)) return "success";
  if (/quality|broken/i.test(reason)) return "warning";
  return "secondary";
}

export const TombstoneRow: React.FC<{
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

  const editLabel = intl.formatMessage({ id: "tombstones.edit_reason" });
  const deleteLabel = intl.formatMessage({ id: "tombstones.delete_title" });

  return (
    <div role="rowgroup">
      <div role="row" className={cx("tombstone-grid tombstone-row", { open })}>
        <div role="cell">
          <ExpandCollapseButton
            collapsed={!open}
            setCollapsed={onToggle}
            collapsedIcon={faChevronRight}
            notCollapsedIcon={faChevronDown}
          />
        </div>
        <div role="cell">
          {/* without a title the filename stands in, styled as untitled */}
          <Button
            variant="link"
            className={cx("tombstone-title", { untitled: !title })}
            title={title || file?.basename}
            aria-expanded={open}
            onClick={onToggle}
          >
            {title || file?.basename || (
              <FormattedMessage id="tombstones.untitled" />
            )}
          </Button>
          <TruncatedText className="tombstone-sub text-muted" text={meta} />
        </div>
        <div role="cell">
          <div>{TextUtils.formatDate(intl, tombstone.deleted_at, false)}</div>
          <div className="tombstone-sub text-muted">
            {intl.formatTime(tombstone.deleted_at)}
          </div>
        </div>
        <div role="cell">
          <Badge
            pill
            variant={manual ? "secondary" : "primary"}
            title={intl.formatMessage({
              id: manual
                ? "tombstones.origin_manual"
                : "tombstones.origin_clean",
            })}
          >
            <Icon icon={manual ? faUser : faBolt} />{" "}
            <FormattedMessage
              id={
                manual
                  ? "tombstones.origin_manual_short"
                  : "tombstones.origin_clean_short"
              }
            />
          </Badge>
        </div>
        <div role="cell">
          {tombstone.reason ? (
            <Badge
              pill
              variant={reasonVariant(tombstone.reason)}
              className="tombstone-reason-badge"
              title={tombstone.reason}
            >
              {tombstone.reason}
            </Badge>
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
          {file ? <TruncatedText text={file.basename} /> : "—"}
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
            title={editLabel}
            aria-label={editLabel}
            onClick={onEdit}
          >
            <Icon icon={faPencilAlt} />
          </Button>
          <Button
            variant="danger"
            title={deleteLabel}
            aria-label={deleteLabel}
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
