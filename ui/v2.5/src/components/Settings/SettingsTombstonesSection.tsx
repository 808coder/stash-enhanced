import React from "react";
import { useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { evictQueries, useConfiguration } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import { LoadingIndicator } from "../Shared/LoadingIndicator";
import { TagIDSelect } from "../Tags/TagSelect";
import { SettingSection } from "./SettingSection";
import { BooleanSetting, Setting, StringListSetting } from "./Inputs";

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

  // unset fields are left unchanged by the server; an empty
  // previouslyDeletedTagId clears the tag
  async function save(input: GQL.ConfigTombstonesInput) {
    try {
      await configure({ variables: { input } });
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
      <StringListSetting
        id="tombstones-reasons"
        headingID="config.tombstones.deletion_reasons"
        subHeadingID="config.tombstones.deletion_reasons_description"
        value={config.reasons}
        onChange={(v) => save({ reasons: v })}
      />
      <Setting
        id="tombstones-previously-deleted-tag"
        headingID="config.tombstones.previously_deleted_tag"
        subHeadingID="config.tombstones.previously_deleted_tag_description"
      >
        <TagIDSelect
          ids={
            config.previouslyDeletedTagId ? [config.previouslyDeletedTagId] : []
          }
          onSelect={(items) =>
            save({ previouslyDeletedTagId: items[0]?.id ?? "" })
          }
          noSelectionString={intl.formatMessage({
            id: "config.tombstones.previously_deleted_tag_default",
          })}
          // render the menu in a portal so the settings card does not clip it
          menuPortalTarget={document.body}
        />
      </Setting>
    </SettingSection>
  );
};
