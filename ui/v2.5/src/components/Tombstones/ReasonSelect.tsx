import React from "react";
import { Form } from "react-bootstrap";
import { useIntl } from "react-intl";
import { useConfigurationContext } from "src/hooks/Config";

// Select for a tombstone deletion reason. The empty value means "no reason".
// A current value that is not in the configured list is still offered, so
// editing an old tombstone does not silently change its reason.
export const ReasonSelect: React.FC<{
  id?: string;
  value: string;
  onChange: (value: string) => void;
}> = ({ id, value, onChange }) => {
  const intl = useIntl();
  const { configuration: config } = useConfigurationContext();

  const reasons = (config?.tombstones.reasons ?? []).map((r) => r.name);
  const options =
    value && !reasons.includes(value) ? [value, ...reasons] : reasons;

  return (
    <Form.Control
      as="select"
      id={id}
      className="input-control"
      value={value}
      onChange={(e) => onChange(e.currentTarget.value)}
    >
      <option value="">
        {intl.formatMessage({ id: "tombstones.no_reason" })}
      </option>
      {options.map((r) => (
        <option key={r} value={r}>
          {r}
        </option>
      ))}
    </Form.Control>
  );
};
