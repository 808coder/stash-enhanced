import React from "react";
import { useIntl } from "react-intl";

const MiB = 1024 * 1024;
const GiB = 1024 * MiB;

// Always in whole MiB so sizes compare at a glance; GiB on hover.
export const TombstoneFileSize: React.FC<{ size: number }> = ({ size }) => {
  const intl = useIntl();
  const bytes = Number.isFinite(size) ? size : 0;

  const gib = intl.formatNumber(bytes / GiB, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });

  return (
    <span title={`${gib} GiB`}>
      {intl.formatNumber(Math.round(bytes / MiB))} MiB
    </span>
  );
};
