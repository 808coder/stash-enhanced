import React, { useState } from "react";
import { Button } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { useTranslationEnabled } from "src/components/Scenes/SceneTranslation";
import { SettingSection } from "../SettingSection";
import { BooleanSetting, SettingGroup } from "../Inputs";

/** Task that translates all scenes with a mapped language tag. */
export const TranslateMetadataTask: React.FC = () => {
  const intl = useIntl();
  const Toast = useToast();
  const enabled = useTranslationEnabled();
  const [force, setForce] = useState(false);
  const [translate] = GQL.useMetadataTranslateMutation();

  if (!enabled) return null;

  async function onTranslate() {
    try {
      await translate({ variables: { input: { force } } });
      Toast.success(
        intl.formatMessage(
          { id: "config.tasks.added_job_to_queue" },
          {
            operation_name: intl.formatMessage({
              id: "translation.task.heading",
            }),
          }
        )
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <SettingSection>
      <SettingGroup
        settingProps={{
          headingID: "translation.task.heading",
          subHeadingID: "translation.task.description",
        }}
        topLevel={
          <Button variant="secondary" type="submit" onClick={onTranslate}>
            <FormattedMessage id="translation.actions.translate" />
          </Button>
        }
      >
        <BooleanSetting
          id="translate-metadata-force"
          headingID="translation.task.force"
          subHeadingID="translation.task.force_description"
          checked={force}
          onChange={setForce}
        />
      </SettingGroup>
    </SettingSection>
  );
};
