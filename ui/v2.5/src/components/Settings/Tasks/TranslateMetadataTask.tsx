import React, { useState } from "react";
import { Button } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { useTranslationEnabled } from "src/components/Scenes/SceneTranslation";
import { SettingSection } from "../SettingSection";
import { BooleanSetting, SettingGroup } from "../Inputs";

/**
 * Task that translates all scenes with a mapped language tag and the names
 * of tags in a mapped language.
 */
export const TranslateMetadataTask: React.FC = () => {
  const intl = useIntl();
  const Toast = useToast();
  const enabled = useTranslationEnabled();
  const [scenes, setScenes] = useState(true);
  const [force, setForce] = useState(false);
  const [tags, setTags] = useState(false);
  const [mergeOnCollision, setMergeOnCollision] = useState(true);
  const [translate] = GQL.useMetadataTranslateMutation();

  if (!enabled) return null;

  async function onTranslate() {
    try {
      await translate({
        variables: { input: { scenes, force, tags, mergeOnCollision } },
      });
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
          <Button
            variant="secondary"
            type="submit"
            disabled={!scenes && !tags}
            onClick={onTranslate}
          >
            <FormattedMessage id="translation.actions.translate" />
          </Button>
        }
      >
        <BooleanSetting
          id="translate-metadata-scenes"
          headingID="translation.task.scenes"
          subHeadingID="translation.task.scenes_description"
          checked={scenes}
          onChange={setScenes}
        />
        <BooleanSetting
          id="translate-metadata-force"
          headingID="translation.task.force"
          subHeadingID="translation.task.force_description"
          checked={force}
          disabled={!scenes}
          onChange={setForce}
        />
        <BooleanSetting
          id="translate-metadata-tags"
          headingID="translation.task.tags"
          subHeadingID="translation.task.tags_description"
          checked={tags}
          onChange={setTags}
        />
        <BooleanSetting
          id="translate-metadata-merge-on-collision"
          headingID="translation.task.merge_on_collision"
          subHeadingID="translation.task.merge_on_collision_description"
          checked={mergeOnCollision}
          disabled={!tags}
          onChange={setMergeOnCollision}
        />
      </SettingGroup>
    </SettingSection>
  );
};
