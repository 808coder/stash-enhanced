import React from "react";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { evictQueries, useConfiguration } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import { LoadingIndicator } from "../Shared/LoadingIndicator";
import { TagIDSelect } from "../Tags/TagSelect";
import { SettingSection } from "./SettingSection";
import { BooleanSetting } from "./Inputs";

export const SettingsTombstonesSection: React.FC = () => {
  const intl = useIntl();
  const Toast = useToast();
  const { data, loading, error } = useConfiguration();
  const [configure] = GQL.useConfigureTombstonesMutation({
    update(cache, result) {
      if (!result.data?.configureTombstones) return;
      evictQueries(cache, [GQL.ConfigurationDocument]);
    },
  });

  if (error) return <h1>{error.message}</h1>;
  if (loading || !data) return <LoadingIndicator />;

  const config = data.configuration.tombstones;

  // the server clears the tag when previouslyDeletedTagId is empty, so the
  // current value is always sent along
  async function save(input: Partial<GQL.ConfigTombstonesInput>) {
    try {
      await configure({
        variables: {
          input: {
            record: config.record,
            previouslyDeletedTagId: config.previouslyDeletedTagId ?? "",
            ...input,
          },
        },
      });
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <SettingSection headingID="config.tombstones.title">
      <BooleanSetting
        id="tombstones-record"
        headingID="config.tombstones.record"
        subHeadingID="config.tombstones.record_description"
        checked={config.record}
        onChange={(v) => save({ record: v })}
      />
      <div className="setting">
        <div>
          <h3>
            <FormattedMessage id="config.tombstones.previously_deleted_tag" />
          </h3>
          <div className="sub-heading">
            <FormattedMessage id="config.tombstones.previously_deleted_tag_description" />
          </div>
        </div>
        <div>
          <TagIDSelect
            ids={
              config.previouslyDeletedTagId
                ? [config.previouslyDeletedTagId]
                : []
            }
            onSelect={(items) =>
              save({ previouslyDeletedTagId: items[0]?.id ?? "" })
            }
            noSelectionString={intl.formatMessage({
              id: "config.tombstones.previously_deleted_tag_default",
            })}
          />
        </div>
      </div>
    </SettingSection>
  );
};
