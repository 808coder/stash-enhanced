import React from "react";
import { Badge } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { faBolt, faUser } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { useConfigurationContext } from "src/hooks/Config";
import { Icon } from "../Shared/Icon";

// Theme variant of each reason colour.
export const reasonColorVariant: Record<GQL.TombstoneReasonColor, string> = {
  [GQL.TombstoneReasonColor.Neutral]: "secondary",
  [GQL.TombstoneReasonColor.Blue]: "primary",
  [GQL.TombstoneReasonColor.Teal]: "info",
  [GQL.TombstoneReasonColor.Green]: "success",
  [GQL.TombstoneReasonColor.Yellow]: "warning",
  [GQL.TombstoneReasonColor.Red]: "danger",
};

// The colour of a Tombstone's frozen reason text, looked up in the configured
// reasons ignoring case. A reason that is no longer configured (renamed or
// removed) is neutral.
export function reasonColor(
  reason: string,
  reasons: readonly Pick<GQL.TombstoneReason, "name" | "color">[]
): GQL.TombstoneReasonColor {
  const key = reason.trim().toLowerCase();
  const found = reasons.find((r) => r.name.trim().toLowerCase() === key);
  return found?.color ?? GQL.TombstoneReasonColor.Neutral;
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

// Truncated to one line; the full reason is in the title.
// The colour is looked up in the configured reasons unless given.
export const ReasonBadge: React.FC<{
  reason: string;
  color?: GQL.TombstoneReasonColor;
}> = ({ reason, color }) => {
  const { configuration } = useConfigurationContext();
  const c =
    color ?? reasonColor(reason, configuration?.tombstones.reasons ?? []);

  return (
    <Badge
      pill
      variant={reasonColorVariant[c]}
      className="tombstone-reason-badge"
      title={reason}
    >
      {reason}
    </Badge>
  );
};
