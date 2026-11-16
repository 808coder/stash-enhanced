import React, { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { faPencilAlt, faTrash } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { ModalComponent } from "../Shared/Modal";
import { ReasonSelect } from "./ReasonSelect";
import { TombstoneData } from "./types";

export const EditReasonDialog: React.FC<{
  tombstone: TombstoneData;
  onClose: (saved: boolean) => void;
}> = ({ tombstone, onClose }) => {
  const intl = useIntl();
  const Toast = useToast();
  const [reason, setReason] = useState(tombstone.reason);
  const [saving, setSaving] = useState(false);
  const [update] = GQL.useTombstoneUpdateMutation();

  async function onSave() {
    setSaving(true);
    try {
      await update({
        variables: { input: { id: tombstone.id, reason: reason.trim() } },
      });
      Toast.success(intl.formatMessage({ id: "tombstones.toast.updated" }));
      onClose(true);
    } catch (e) {
      Toast.error(e);
      setSaving(false);
    }
  }

  return (
    <ModalComponent
      show
      icon={faPencilAlt}
      header={intl.formatMessage({ id: "tombstones.edit_reason" })}
      accept={{
        onClick: onSave,
        text: intl.formatMessage({ id: "actions.save" }),
      }}
      cancel={{
        onClick: () => onClose(false),
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={saving}
    >
      <ReasonSelect value={reason} onChange={setReason} />
    </ModalComponent>
  );
};

export const DeleteTombstoneDialog: React.FC<{
  tombstone: TombstoneData;
  onClose: (deleted: boolean) => void;
}> = ({ tombstone, onClose }) => {
  const intl = useIntl();
  const Toast = useToast();
  const [deleting, setDeleting] = useState(false);
  const [destroy] = GQL.useTombstoneDestroyMutation();

  async function onDelete() {
    setDeleting(true);
    try {
      await destroy({ variables: { id: tombstone.id } });
      Toast.success(intl.formatMessage({ id: "tombstones.toast.deleted" }));
      onClose(true);
    } catch (e) {
      Toast.error(e);
      setDeleting(false);
    }
  }

  return (
    <ModalComponent
      show
      icon={faTrash}
      header={intl.formatMessage({ id: "tombstones.delete_title" })}
      accept={{
        variant: "danger",
        onClick: onDelete,
        text: intl.formatMessage({ id: "actions.delete" }),
      }}
      cancel={{
        onClick: () => onClose(false),
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={deleting}
    >
      <p>
        <FormattedMessage
          id="tombstones.delete_desc"
          values={{ title: tombstone.title || tombstone.original_title }}
        />
      </p>
    </ModalComponent>
  );
};
