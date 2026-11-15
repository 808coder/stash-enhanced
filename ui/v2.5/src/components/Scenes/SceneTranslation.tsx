import React, { useState } from "react";
import { Dropdown } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import { useConfigurationContext } from "src/hooks/Config";
import { useToast } from "src/hooks/Toast";
import { ScrapeDialog } from "../Shared/ScrapeDialog/ScrapeDialog";
import {
  ScrapedInputGroupRow,
  ScrapedTextAreaRow,
} from "../Shared/ScrapeDialog/ScrapeDialogRow";
import { ScrapeResult } from "../Shared/ScrapeDialog/scrapeResult";

const originalTitleField = "original_title";
const originalDetailsField = "original_details";

/** True if a translation provider is active. */
export function useTranslationEnabled() {
  const { configuration } = useConfigurationContext();
  return configuration.translation.provider !== GQL.TranslationProvider.None;
}

/** True if the scene has original metadata stored by a translation. */
export function hasOriginalMetadata(scene: { custom_fields?: unknown }) {
  const cf = (scene.custom_fields ?? {}) as Record<string, unknown>;
  return (
    cf[originalTitleField] !== undefined ||
    cf[originalDetailsField] !== undefined
  );
}

function useWarningsToast() {
  const Toast = useToast();
  return (warnings: string[]) => {
    if (warnings.length === 0) return;
    Toast.toast({
      variant: "warning",
      content: (
        <ul className="mb-0 pl-3">
          {warnings.map((w) => (
            <li key={w}>{w}</li>
          ))}
        </ul>
      ),
    });
  };
}

interface ISceneTranslateDialogProps {
  sceneId: string;
  translation: GQL.SceneTranslationDataFragment;
  onClose: (saved: boolean) => void;
}

export const SceneTranslateDialog: React.FC<ISceneTranslateDialogProps> = ({
  sceneId,
  translation,
  onClose,
}) => {
  const intl = useIntl();
  const Toast = useToast();
  const [apply] = GQL.useSceneTranslateApplyMutation();

  const [title, setTitle] = useState(
    () =>
      new ScrapeResult<string>(
        translation.title?.current,
        translation.title?.translation
      )
  );
  const [details, setDetails] = useState(
    () =>
      new ScrapeResult<string>(
        translation.details?.current,
        translation.details?.translation
      )
  );

  function fieldInput(
    field: GQL.SceneFieldTranslationDataFragment | null | undefined,
    result: ScrapeResult<string>
  ): GQL.SceneFieldTranslationInput | undefined {
    const value = result.getNewValue();
    if (!field || !result.useNewValue || !value?.trim()) return undefined;
    return {
      source: field.source,
      translation: value,
      language: field.language,
    };
  }

  async function onDialogClose(accept?: boolean) {
    if (!accept) {
      onClose(false);
      return;
    }

    const input: GQL.SceneTranslationApplyInput = {
      id: sceneId,
      provider: translation.provider,
      title: fieldInput(translation.title, title),
      details: fieldInput(translation.details, details),
    };

    if (!input.title && !input.details) {
      onClose(false);
      return;
    }

    try {
      await apply({ variables: { input } });
      Toast.success(intl.formatMessage({ id: "translation.toast.saved" }));
      onClose(true);
    } catch (e) {
      Toast.error(e);
    }
  }

  function renderSource(field?: GQL.SceneFieldTranslationDataFragment | null) {
    // the source is only worth showing when it differs from the current value,
    // i.e. when re-translating from the stored original
    if (!field || field.source === field.current) return undefined;
    return (
      <div className="px-3 pb-2 text-muted small translation-source">
        <FormattedMessage id="translation.original" />: {field.source}
      </div>
    );
  }

  return (
    <ScrapeDialog
      title={intl.formatMessage({ id: "translation.dialog_title" })}
      existingLabel={<FormattedMessage id="translation.current" />}
      scrapedLabel={<FormattedMessage id="translation.translated" />}
      onClose={onDialogClose}
    >
      {translation.warnings.length > 0 ? (
        <ul className="px-3 text-warning small">
          {translation.warnings.map((w) => (
            <li key={w}>{w}</li>
          ))}
        </ul>
      ) : undefined}
      {translation.title ? (
        <>
          <ScrapedInputGroupRow
            title={intl.formatMessage({ id: "title" })}
            field="title"
            result={title}
            onChange={setTitle}
          />
          {renderSource(translation.title)}
        </>
      ) : undefined}
      {translation.details ? (
        <>
          <ScrapedTextAreaRow
            title={intl.formatMessage({ id: "details" })}
            field="details"
            result={details}
            onChange={setDetails}
          />
          {renderSource(translation.details)}
        </>
      ) : undefined}
    </ScrapeDialog>
  );
};

/**
 * Translate and restore actions for the operations menu of a scene, plus the
 * review dialog they open.
 */
export function useSceneTranslation(scene: GQL.SceneDataFragment) {
  const intl = useIntl();
  const Toast = useToast();
  const showWarnings = useWarningsToast();
  const enabled = useTranslationEnabled();
  const { configuration } = useConfigurationContext();
  const applyWithoutReview = configuration.translation.applyWithoutReview;

  const [preview] = GQL.useSceneTranslatePreviewMutation();
  const [translateAndSave] = GQL.useSceneTranslateMutation();
  const [restore] = GQL.useScenesRestoreOriginalMutation();

  const [busy, setBusy] = useState(false);
  const [pending, setPending] = useState<GQL.SceneTranslationDataFragment>();

  const original = hasOriginalMetadata(scene);

  async function onTranslate(force: boolean) {
    setBusy(true);
    Toast.success(intl.formatMessage({ id: "translation.toast.translating" }));
    try {
      if (applyWithoutReview) {
        const result = await translateAndSave({
          variables: { id: scene.id, force },
        });
        const t = result.data?.sceneTranslate;
        if (!t) return;
        showWarnings(t.warnings);
        Toast.success(
          intl.formatMessage({
            id: t.saved
              ? "translation.toast.saved"
              : "translation.toast.nothing_to_translate",
          })
        );
        return;
      }

      const result = await preview({ variables: { id: scene.id, force } });
      const t = result.data?.sceneTranslatePreview;
      if (!t) return;
      if (!t.title && !t.details) {
        showWarnings(t.warnings);
        Toast.success(
          intl.formatMessage({ id: "translation.toast.nothing_to_translate" })
        );
        return;
      }
      setPending(t);
    } catch (e) {
      Toast.error(e);
    } finally {
      setBusy(false);
    }
  }

  async function onRestore() {
    try {
      await restore({ variables: { ids: [scene.id] } });
      Toast.success(intl.formatMessage({ id: "translation.toast.restored" }));
    } catch (e) {
      Toast.error(e);
    }
  }

  const menuItems = enabled ? (
    <>
      <Dropdown.Item
        key="translate"
        className="bg-secondary text-white"
        disabled={busy}
        onClick={() => onTranslate(false)}
      >
        <FormattedMessage id="translation.actions.translate" />
        {applyWithoutReview ? undefined : "…"}
      </Dropdown.Item>
      {original ? (
        <>
          <Dropdown.Item
            key="translate-again"
            className="bg-secondary text-white"
            disabled={busy}
            onClick={() => onTranslate(true)}
          >
            <FormattedMessage id="translation.actions.translate_again" />
            {applyWithoutReview ? undefined : "…"}
          </Dropdown.Item>
          <Dropdown.Item
            key="restore-original"
            className="bg-secondary text-white"
            onClick={() => onRestore()}
          >
            <FormattedMessage id="translation.actions.restore_original" />
          </Dropdown.Item>
        </>
      ) : undefined}
    </>
  ) : undefined;

  const dialog = pending ? (
    <SceneTranslateDialog
      sceneId={scene.id}
      translation={pending}
      onClose={() => setPending(undefined)}
    />
  ) : undefined;

  return { menuItems, dialog };
}

/** Bulk translate and restore operations for scene lists. */
export function useSceneTranslationOperations(
  getSelectedIds: () => string[],
  hasSelection: boolean
) {
  const intl = useIntl();
  const Toast = useToast();
  const enabled = useTranslationEnabled();
  const [translate] = GQL.useMetadataTranslateMutation();
  const [restore] = GQL.useScenesRestoreOriginalMutation();

  async function onTranslate() {
    try {
      await translate({ variables: { input: { sceneIds: getSelectedIds() } } });
      Toast.success(
        intl.formatMessage(
          { id: "config.tasks.added_job_to_queue" },
          {
            operation_name: intl.formatMessage({
              id: "translation.actions.translate",
            }),
          }
        )
      );
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onRestore() {
    try {
      await restore({ variables: { ids: getSelectedIds() } });
      Toast.success(intl.formatMessage({ id: "translation.toast.restored" }));
    } catch (e) {
      Toast.error(e);
    }
  }

  return [
    {
      text: intl.formatMessage({ id: "translation.actions.translate" }),
      onClick: onTranslate,
      isDisplayed: () => enabled && hasSelection,
    },
    {
      text: intl.formatMessage({ id: "translation.actions.restore_original" }),
      onClick: onRestore,
      isDisplayed: () => enabled && hasSelection,
    },
  ];
}
