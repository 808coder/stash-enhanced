import React, { useState } from "react";
import { Button, ButtonGroup, Card, Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import cx from "classnames";
import { faArrowDown, faArrowUp } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { useDebouncedState } from "src/hooks/debounce";
import { ClearableInput } from "../Shared/ClearableInput";
import { ErrorMessage } from "../Shared/ErrorMessage";
import { Icon } from "../Shared/Icon";
import { LoadingIndicator } from "../Shared/LoadingIndicator";
import { PageSizeSelector } from "../List/ListFilter";
import { Pagination, PaginationIndex } from "../List/Pagination";
import { DeleteTombstoneDialog, EditReasonDialog } from "./TombstoneDialogs";
import { TombstoneRow } from "./TombstoneRow";
import { TombstoneData } from "./types";

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

// Keep in sync with models.AllTombstoneSort and the findTombstones sort
// docstring in graphql/schema/types/tombstone.graphql.
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
  // the default page size of the other lists
  const [pageSize, setPageSize] = useState(40);
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
            <PaginationIndex
              itemsPerPage={pageSize}
              currentPage={page}
              totalItems={count}
            />
            <PageSizeSelector
              pageSize={pageSize}
              setPageSize={(size) => {
                setPageSize(size);
                setPage(1);
              }}
            />
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
