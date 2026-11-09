import React, { useState } from "react";
import { Form } from "react-bootstrap";
import { FormattedMessage } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { PerformerCard } from "src/components/Performers/PerformerCard";
import { sortPerformers } from "src/core/performers";

type Performer = Parameters<typeof PerformerCard>[0]["performer"];

interface IDetailPanelPerformersProps {
  performers: Performer[];
  ageFromDate?: string;
  className: string;
}

const isMale = (p: Performer) => p.gender === GQL.GenderEnum.Male;

/**
 * Performers section of the scene/gallery/image detail panels, with a
 * toggle to hide male performers. The toggle state is intentionally not
 * persisted; use `key` on the component to reset it per entity.
 */
export const DetailPanelPerformers: React.FC<IDetailPanelPerformersProps> = ({
  performers,
  ageFromDate,
  className,
}) => {
  const [hideMale, setHideMale] = useState(false);

  if (performers.length === 0) return null;

  const maleCount = performers.filter(isMale).length;
  const hiding = hideMale && maleCount > 0;
  const visible = sortPerformers(
    hiding ? performers.filter((p) => !isMale(p)) : performers
  );

  return (
    <>
      <div className="d-flex align-items-center justify-content-between">
        <h6>
          <FormattedMessage
            id="countables.performers"
            values={{ count: visible.length }}
          />
          {hiding && (
            <>
              {" "}
              <FormattedMessage
                id="performers_hidden_count"
                values={{ count: maleCount }}
              />
            </>
          )}
        </h6>
        {maleCount > 0 && (
          <Form.Check
            type="switch"
            id={`${className}-hide-male`}
            checked={hideMale}
            onChange={() => setHideMale(!hideMale)}
            label={<FormattedMessage id="hide_male_performers" />}
          />
        )}
      </div>
      <div className={`row justify-content-center ${className}`}>
        {visible.map((performer) => (
          <PerformerCard
            key={performer.id}
            performer={performer}
            ageFromDate={ageFromDate}
          />
        ))}
      </div>
    </>
  );
};
