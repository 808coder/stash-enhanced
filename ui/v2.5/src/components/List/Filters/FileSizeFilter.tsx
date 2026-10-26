import React, { useState } from "react";
import { Form } from "react-bootstrap";
import { useIntl } from "react-intl";
import { CriterionModifier } from "src/core/generated-graphql";
import {
  FILE_SIZE_GIB,
  FILE_SIZE_MIB,
  FileSizeCriterion,
} from "src/models/list-filter/criteria/criterion";
import { INumberValue } from "src/models/list-filter/types";
import { NumberField } from "src/utils/form";

interface IFileSizeFilterProps {
  criterion: FileSizeCriterion;
  onValueChanged: (value: INumberValue) => void;
}

const units = [
  { label: "MiB", bytes: FILE_SIZE_MIB },
  { label: "GiB", bytes: FILE_SIZE_GIB },
];

function bytesToText(bytes: number | undefined, unitBytes: number) {
  if (bytes === undefined) return "";
  return String(Number((bytes / unitBytes).toFixed(6)));
}

function textToBytes(text: string, unitBytes: number) {
  const n = parseFloat(text);
  if (Number.isNaN(n)) return undefined;
  return Math.round(n * unitBytes);
}

export const FileSizeFilter: React.FC<IFileSizeFilterProps> = ({
  criterion,
  onValueChanged,
}) => {
  const intl = useIntl();

  const { value } = criterion;

  // the unit is only used for display - the criterion value is always in bytes
  const [unitBytes, setUnitBytes] = useState(
    (value?.value ?? 0) >= FILE_SIZE_GIB ? FILE_SIZE_GIB : FILE_SIZE_MIB
  );
  const [texts, setTexts] = useState({
    value: bytesToText(value?.value, unitBytes),
    value2: bytesToText(value?.value2, unitBytes),
  });

  function onChanged(text: string, property: "value" | "value2") {
    setTexts({ ...texts, [property]: text });
    onValueChanged({
      ...value,
      [property]: textToBytes(text, unitBytes),
    });
  }

  function onUnitChanged(newUnitBytes: number) {
    setUnitBytes(newUnitBytes);
    onValueChanged({
      value: textToBytes(texts.value, newUnitBytes),
      value2: textToBytes(texts.value2, newUnitBytes),
    });
  }

  const isBetween =
    criterion.modifier === CriterionModifier.Between ||
    criterion.modifier === CriterionModifier.NotBetween;

  let placeholder: string;
  if (criterion.modifier === CriterionModifier.GreaterThan || isBetween) {
    placeholder = intl.formatMessage({ id: "criterion.greater_than" });
  } else if (criterion.modifier === CriterionModifier.LessThan) {
    placeholder = intl.formatMessage({ id: "criterion.less_than" });
  } else {
    placeholder = intl.formatMessage({ id: "criterion.value" });
  }

  return (
    <>
      <Form.Group>
        <NumberField
          className="btn-secondary"
          min={0}
          step="any"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            onChanged(e.target.value, "value")
          }
          value={texts.value}
          placeholder={placeholder}
        />
      </Form.Group>
      {isBetween && (
        <Form.Group>
          <NumberField
            className="btn-secondary"
            min={0}
            step="any"
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              onChanged(e.target.value, "value2")
            }
            value={texts.value2}
            placeholder={intl.formatMessage({ id: "criterion.less_than" })}
          />
        </Form.Group>
      )}
      <Form.Group>
        <Form.Control
          as="select"
          className="btn-secondary"
          value={unitBytes}
          onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
            onUnitChanged(Number(e.target.value))
          }
        >
          {units.map((u) => (
            <option key={u.label} value={u.bytes}>
              {u.label}
            </option>
          ))}
        </Form.Control>
      </Form.Group>
    </>
  );
};
