import React, { useEffect, useState } from "react";
import { Button, Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { faPlus, faTrash } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { evictQueries, useConfiguration } from "src/core/StashService";
import { useToast } from "src/hooks/Toast";
import { Icon } from "../Shared/Icon";
import { LoadingIndicator } from "../Shared/LoadingIndicator";
import { TagIDSelect } from "../Tags/TagSelect";
import { SettingSection } from "./SettingSection";
import {
  BooleanSetting,
  ModalSetting,
  SelectSetting,
  Setting,
  StringSetting,
} from "./Inputs";

const languageCodeRE = /^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$/;

function isValidLanguageCode(code: string) {
  return languageCodeRE.test(code.trim());
}

const providerMessageIDs: Record<GQL.TranslationProvider, string> = {
  [GQL.TranslationProvider.None]: "config.translation.provider_none",
  [GQL.TranslationProvider.OpenaiCompatible]:
    "config.translation.provider_openai_compatible",
};

interface IMappingRow {
  tagId: string;
  languageCode: string;
}

function isCompleteRow(r: IMappingRow) {
  return r.tagId !== "" && isValidLanguageCode(r.languageCode);
}

function toInput(rows: IMappingRow[]): GQL.TranslationLanguageMappingInput[] {
  return rows.map((r) => ({
    tagId: r.tagId,
    languageCode: r.languageCode.trim(),
  }));
}

const LanguageMappingEditor: React.FC<{
  mappings: GQL.ConfigTranslationDataFragment["languageMappings"];
  onSave: (mappings: GQL.TranslationLanguageMappingInput[]) => void;
}> = ({ mappings, onSave }) => {
  const intl = useIntl();
  const [rows, setRows] = useState<IMappingRow[]>([]);

  // reset local rows whenever the saved mappings change
  useEffect(() => {
    setRows(
      mappings.map((m) => ({ tagId: m.tagId, languageCode: m.languageCode }))
    );
  }, [mappings]);

  function update(newRows: IMappingRow[]) {
    setRows(newRows);
    // only complete rows are saved; incomplete rows stay local until finished
    if (newRows.every(isCompleteRow)) {
      onSave(toInput(newRows));
    }
  }

  function setRow(index: number, row: Partial<IMappingRow>) {
    const newRows = rows.map((r, i) => (i === index ? { ...r, ...row } : r));
    update(newRows);
  }

  const incomplete = rows.some((r) => !isCompleteRow(r));

  return (
    <div className="setting translation-language-mappings">
      <div className="w-100">
        {rows.map((row, index) => (
          <div key={index} className="d-flex align-items-center mb-2">
            <div className="flex-grow-1 mr-2">
              <TagIDSelect
                ids={row.tagId ? [row.tagId] : []}
                onSelect={(items) =>
                  setRow(index, { tagId: items[0]?.id ?? "" })
                }
                noSelectionString={intl.formatMessage({
                  id: "config.translation.language_mappings.tag",
                })}
              />
            </div>
            <Form.Control
              className="text-input mr-2 translation-language-code"
              placeholder={intl.formatMessage({
                id: "config.translation.language_mappings.language_code",
              })}
              value={row.languageCode}
              isInvalid={
                row.languageCode !== "" &&
                !isValidLanguageCode(row.languageCode)
              }
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setRow(index, { languageCode: e.currentTarget.value })
              }
            />
            <Button
              variant="danger"
              title={intl.formatMessage({ id: "actions.remove" })}
              onClick={() => update(rows.filter((_, i) => i !== index))}
            >
              <Icon icon={faTrash} />
            </Button>
          </div>
        ))}
        {incomplete ? (
          <div className="sub-heading text-warning mb-2">
            <FormattedMessage id="config.translation.language_mappings.incomplete" />
          </div>
        ) : undefined}
        <Button
          variant="secondary"
          onClick={() => setRows([...rows, { tagId: "", languageCode: "" }])}
        >
          <Icon icon={faPlus} />{" "}
          <FormattedMessage id="config.translation.language_mappings.add" />
        </Button>
      </div>
    </div>
  );
};

const TestConnectionSetting: React.FC = () => {
  const intl = useIntl();
  const [testTranslation, { data, loading, error }] =
    GQL.useTestTranslationLazyQuery({ fetchPolicy: "network-only" });

  const result = data?.testTranslation;

  function renderResult() {
    if (loading) return <LoadingIndicator inline small message="" />;
    if (error) {
      return (
        <span className="text-danger">
          {intl.formatMessage(
            { id: "config.translation.test_failed" },
            { error: error.message }
          )}
        </span>
      );
    }
    if (!result) return undefined;
    if (!result.success) {
      return (
        <span className="text-danger">
          {intl.formatMessage(
            { id: "config.translation.test_failed" },
            { error: result.error ?? "" }
          )}
        </span>
      );
    }
    return (
      <span className="text-success">
        {result.source} → {result.translation}
      </span>
    );
  }

  return (
    <Setting
      headingID="config.translation.test_connection"
      subHeadingID="config.translation.test_connection_description"
    >
      <div className="d-flex align-items-center">
        <div className="mr-3">{renderResult()}</div>
        <Button
          disabled={loading}
          onClick={() => testTranslation({ variables: { input: {} } })}
        >
          <FormattedMessage id="config.translation.test_connection" />
        </Button>
      </div>
    </Setting>
  );
};

export const SettingsTranslationPanel: React.FC = () => {
  const intl = useIntl();
  const Toast = useToast();
  const { data, loading, error } = useConfiguration();
  const [configure] = GQL.useConfigureTranslationMutation({
    update(cache, result) {
      if (!result.data?.configureTranslation) return;
      evictQueries(cache, [GQL.ConfigurationDocument]);
    },
  });

  if (error) return <h1>{error.message}</h1>;
  if (loading || !data) return <LoadingIndicator />;

  const config = data.configuration.translation;
  const { openai } = config;

  async function save(input: GQL.ConfigTranslationInput) {
    try {
      await configure({ variables: { input } });
    } catch (e) {
      Toast.error(e);
    }
  }

  function validateLanguageCode(v: string) {
    if (!isValidLanguageCode(v)) {
      throw new Error(
        intl.formatMessage({ id: "config.translation.invalid_language_code" })
      );
    }
  }

  const notSet = (
    <span className="text-muted">
      <FormattedMessage id="config.translation.not_set" />
    </span>
  );

  return (
    <>
      <SettingSection headingID="config.translation.title">
        <SelectSetting
          id="translation-provider"
          headingID="config.translation.provider"
          subHeadingID="config.translation.provider_description"
          value={config.provider}
          onChange={(v) => save({ provider: v as GQL.TranslationProvider })}
        >
          {Object.values(GQL.TranslationProvider).map((p) => (
            <option key={p} value={p}>
              {intl.formatMessage({ id: providerMessageIDs[p] })}
            </option>
          ))}
        </SelectSetting>

        <ModalSetting<string>
          id="translation-target-language"
          headingID="config.translation.target_language"
          subHeadingID="config.translation.target_language_description"
          value={config.targetLanguage}
          validateChange={validateLanguageCode}
          onChange={(v) => save({ targetLanguage: v.trim() })}
          renderField={(value, setValue) => (
            <Form.Control
              className="text-input"
              value={value ?? ""}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setValue(e.currentTarget.value)
              }
            />
          )}
          renderValue={(value) => <span>{value}</span>}
        />

        <BooleanSetting
          id="translation-apply-without-review"
          headingID="config.translation.apply_without_review"
          subHeadingID="config.translation.apply_without_review_description"
          checked={config.applyWithoutReview}
          onChange={(v) => save({ applyWithoutReview: v })}
        />

        <TestConnectionSetting />
      </SettingSection>

      <SettingSection
        headingID="config.translation.openai.title"
        subHeadingID="config.translation.openai.description"
      >
        <StringSetting
          id="translation-openai-base-url"
          headingID="config.translation.openai.base_url"
          subHeadingID="config.translation.openai.base_url_description"
          value={openai.baseURL}
          onChange={(v) => save({ openai: { baseURL: v.trim() } })}
        />

        <StringSetting
          id="translation-openai-model"
          headingID="config.translation.openai.model"
          subHeadingID="config.translation.openai.model_description"
          value={openai.model}
          onChange={(v) => save({ openai: { model: v.trim() } })}
        />

        <ModalSetting<string>
          id="translation-openai-api-key"
          headingID="config.translation.openai.api_key"
          subHeadingID="config.translation.openai.api_key_description"
          value={openai.apiKey}
          onChange={(v) => save({ openai: { apiKey: v.trim() } })}
          renderField={(value, setValue) => (
            <Form.Control
              className="text-input"
              type="password"
              autoComplete="new-password"
              value={value ?? ""}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setValue(e.currentTarget.value)
              }
            />
          )}
          renderValue={(value) => (value ? <span>••••••••</span> : notSet)}
        />

        <ModalSetting<string>
          id="translation-openai-system-prompt"
          headingID="config.translation.openai.system_prompt"
          subHeadingID="config.translation.openai.system_prompt_description"
          value={openai.systemPrompt || config.defaultSystemPrompt}
          modalProps={{ size: "lg" }}
          onChange={(v) =>
            // storing the default verbatim would freeze it; store empty instead
            save({
              openai: {
                systemPrompt:
                  v.trim() === config.defaultSystemPrompt.trim() ? "" : v,
              },
            })
          }
          renderField={(value, setValue) => (
            <>
              <Form.Control
                as="textarea"
                className="text-input"
                rows={16}
                value={value ?? ""}
                onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
                  setValue(e.currentTarget.value)
                }
              />
              <Button
                variant="secondary"
                className="mt-2"
                onClick={() => setValue(config.defaultSystemPrompt)}
              >
                <FormattedMessage id="config.translation.openai.reset_to_default" />
              </Button>
            </>
          )}
          renderValue={() => (
            <span>
              <FormattedMessage
                id={
                  openai.systemPrompt
                    ? "config.translation.openai.prompt_custom"
                    : "config.translation.openai.prompt_default"
                }
              />
            </span>
          )}
        />
      </SettingSection>

      <SettingSection
        headingID="config.translation.language_mappings.title"
        subHeadingID="config.translation.language_mappings.description"
      >
        <LanguageMappingEditor
          mappings={config.languageMappings}
          onSave={(languageMappings) => save({ languageMappings })}
        />
      </SettingSection>
    </>
  );
};
