import React from "react";
import { Badge } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import cx from "classnames";
import { faBolt, faUser } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "../Shared/Icon";

// Reasons are free text from a configurable list, so the colour is a best
// effort on the default reasons and neutral for everything else.
function reasonVariant(reason: string) {
  if (/duplicate/i.test(reason)) return "info";
  if (/replaced/i.test(reason)) return "success";
  if (/quality|broken/i.test(reason)) return "warning";
  return "secondary";
}

export const OriginBadge: React.FC<{ origin: GQL.TombstoneOrigin }> = ({
  origin,
}) => {
  const intl = useIntl();
  const manual = origin === GQL.TombstoneOrigin.Manual;

  return (
    <Badge
      pill
      variant={manual ? "secondary" : "primary"}
      title={intl.formatMessage({
        id: manual ? "tombstones.origin_manual" : "tombstones.origin_clean",
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
  );
};

// Truncated to one line in the table row, wrapped in full in the details.
export const ReasonBadge: React.FC<{ reason: string; wrap?: boolean }> = ({
  reason,
  wrap,
}) => (
  <Badge
    pill
    variant={reasonVariant(reason)}
    className={cx("tombstone-reason-badge", { wrap })}
    title={wrap ? undefined : reason}
  >
    {reason}
  </Badge>
);
