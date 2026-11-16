import React, { useState } from "react";
import { Button, Form, InputGroup } from "react-bootstrap";
import { useIntl } from "react-intl";
import { faGripVertical, faMinus } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "../Shared/Icon";
import { ReasonBadge } from "../Tombstones/TombstoneBadges";
import { ModalSetting } from "./Inputs";

type Reason = GQL.TombstoneReasonInput;

const colors = Object.values(GQL.TombstoneReasonColor);

// One row per reason with name, colour and a badge preview. Like
// StringListInput, a trailing empty row adds a reason and rows can be
// reordered by dragging.
const TombstoneReasonsInput: React.FC<{
  value: Reason[];
  setValue: (value: Reason[]) => void;
}> = ({ value, setValue }) => {
  const intl = useIntl();
  const [draggedIdx, setDraggedIdx] = useState<number | null>(null);
  const rows = value.concat({
    name: "",
    color: GQL.TombstoneReasonColor.Neutral,
  });
  const last = rows.length - 1;

  function rowChanged(idx: number, row: Reason) {
    const newValues = value.slice();
    newValues[idx] = row;

    // clearing the last reason removes it, as in StringListInput
    if (!row.name && idx === newValues.length - 1) {
      newValues.splice(newValues.length - 1);
    }

    setValue(newValues);
  }

  function handleDragOver(e: React.DragEvent, idx: number) {
    e.dataTransfer.dropEffect = "move";
    e.preventDefault();

    if (draggedIdx === null || draggedIdx === idx || idx === last) return;

    const newValues = value.slice();
    const [dragged] = newValues.splice(draggedIdx, 1);
    newValues.splice(idx, 0, dragged);

    setValue(newValues);
    setDraggedIdx(idx);
  }

  return (
    <div className="string-list-input tombstone-reasons-input">
      <Form.Group>
        {rows.map((r, i) => (
          <InputGroup
            // rows have no stable id; the order is the identity
            key={i}
            onDragOver={(e) => handleDragOver(e, i)}
          >
            <Form.Control
              className="text-input"
              value={r.name}
              aria-label={intl.formatMessage({
                id: "config.tombstones.reason_name",
              })}
              placeholder={
                i === last
                  ? intl.formatMessage({ id: "config.tombstones.add_reason" })
                  : undefined
              }
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                rowChanged(i, { ...r, name: e.currentTarget.value })
              }
            />
            <Form.Control
              as="select"
              className="input-control tombstone-reason-color"
              value={r.color}
              disabled={i === last}
              aria-label={intl.formatMessage({
                id: "config.tombstones.reason_color",
              })}
              onChange={(e) =>
                rowChanged(i, {
                  ...r,
                  color: e.currentTarget.value as GQL.TombstoneReasonColor,
                })
              }
            >
              {colors.map((c) => (
                <option key={c} value={c}>
                  {intl.formatMessage({
                    id: `config.tombstones.reason_colors.${c.toLowerCase()}`,
                  })}
                </option>
              ))}
            </Form.Control>
            <InputGroup.Append>
              <div className="tombstone-reason-preview">
                {r.name.trim() && (
                  <ReasonBadge reason={r.name.trim()} color={r.color} />
                )}
              </div>
              {rows.length > 2 && (
                <Button
                  variant="secondary"
                  className="drag-handle minimal"
                  draggable={i !== last}
                  disabled={i === last}
                  onDragStart={(e) => {
                    e.dataTransfer.dropEffect = "move";
                    setDraggedIdx(i);
                  }}
                  onDragEnd={() => setDraggedIdx(null)}
                >
                  <Icon icon={faGripVertical} />
                </Button>
              )}
              <Button
                variant="danger"
                onClick={() => setValue(value.filter((_v, j) => j !== i))}
                disabled={i === last}
                size="sm"
              >
                <Icon icon={faMinus} />
              </Button>
            </InputGroup.Append>
          </InputGroup>
        ))}
      </Form.Group>
    </div>
  );
};

export const TombstoneReasonsSetting: React.FC<{
  id: string;
  headingID: string;
  subHeadingID: string;
  value: Pick<GQL.TombstoneReason, "name" | "color">[];
  onChange: (value: Reason[]) => void;
}> = ({ value, onChange, ...props }) => {
  const intl = useIntl();

  return (
    <ModalSetting<Reason[]>
      {...props}
      // copy to plain inputs: the query result carries __typename
      value={value.map(({ name, color }) => ({ name, color }))}
      onChange={onChange}
      modalProps={{ size: "lg" }}
      validateChange={(v) => {
        if (!v.some((r) => r.name.trim())) {
          throw new Error(
            intl.formatMessage({ id: "config.tombstones.reasons_required" })
          );
        }
      }}
      renderField={(v, setValue, error) => (
        <>
          <TombstoneReasonsInput value={v ?? []} setValue={setValue} />
          {error && <div className="text-danger mt-2">{error}</div>}
        </>
      )}
      renderValue={(v) => (
        <div className="tombstone-reasons-value">
          {v?.map((r, i) => (
            // rows have no stable id; the order is the identity
            <ReasonBadge key={i} reason={r.name} color={r.color} />
          ))}
        </div>
      )}
    />
  );
};
