import React, { useState } from "react";
import { Button } from "react-bootstrap";
import { FormattedMessage } from "react-intl";
import {
  faChevronDown,
  faChevronRight,
} from "@fortawesome/free-solid-svg-icons";
import { Icon } from "./Icon";

interface ICollapsibleDetailsProps {
  details?: string | null;
}

// Details text of a detail panel, collapsible via its heading. Always starts
// expanded; the state is deliberately not persisted. Give it a `key` of the
// entity id so navigating to another entity resets it.
export const CollapsibleDetails: React.FC<ICollapsibleDetailsProps> = ({
  details,
}) => {
  const [open, setOpen] = useState(true);

  if (!details) return null;

  return (
    <>
      <h6 className="collapsible-details-heading">
        <Button
          variant="link"
          className="minimal collapse-button"
          aria-expanded={open}
          onClick={() => setOpen(!open)}
        >
          <Icon icon={open ? faChevronDown : faChevronRight} fixedWidth />
          <FormattedMessage id="details" />:
        </Button>
      </h6>
      {open && <p className="pre">{details}</p>}
    </>
  );
};
