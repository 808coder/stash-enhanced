import React, { useState } from "react";
import { Button, ButtonGroup, Form, Table } from "react-bootstrap";
import { Link } from "react-router-dom";
import {
  FormattedMessage,
  FormattedNumber,
  FormattedTime,
  useIntl,
} from "react-intl";
import {
  faBox,
  faFileAlt,
  faFilm,
  faImages,
  faMapMarkerAlt,
  faTag,
} from "@fortawesome/free-solid-svg-icons";

import * as GQL from "src/core/generated-graphql";
import SceneQueue from "src/models/sceneQueue";
import TextUtils from "src/utils/text";
import { HoverPopover } from "../Shared/HoverPopover";
import { Icon } from "../Shared/Icon";
import {
  GalleryLink,
  GroupLink,
  SceneMarkerLink,
  TagLink,
} from "../Shared/TagLink";
import { SweatDrops } from "../Shared/SweatDrops";
import { PerformerPopoverButton } from "../Shared/PerformerPopoverButton";
import { FileSize } from "../Shared/FileSize";

const CLASSNAME = "scene-file-details";

function maybeRenderTagPopoverButton(scene: GQL.SlimSceneDataFragment) {
  if (scene.tags.length <= 0) return;

  const popoverContent = scene.tags.map((tag) => (
    <TagLink key={tag.id} tag={tag} />
  ));

  return (
    <HoverPopover placement="bottom" content={popoverContent}>
      <Button className="minimal">
        <Icon icon={faTag} />
        <span>{scene.tags.length}</span>
      </Button>
    </HoverPopover>
  );
}

function maybeRenderPerformerPopoverButton(scene: GQL.SlimSceneDataFragment) {
  if (scene.performers.length <= 0) return;

  return <PerformerPopoverButton performers={scene.performers} />;
}

function maybeRenderGroupPopoverButton(scene: GQL.SlimSceneDataFragment) {
  if (scene.groups.length <= 0) return;

  const popoverContent = scene.groups.map((sceneGroup) => (
    <div className="group-tag-container row" key={sceneGroup.group.id}>
      <Link
        to={`/groups/${sceneGroup.group.id}`}
        className="group-tag col m-auto zoom-2"
      >
        <img
          className="image-thumbnail"
          alt={sceneGroup.group.name ?? ""}
          src={sceneGroup.group.front_image_path ?? ""}
        />
      </Link>
      <GroupLink
        key={sceneGroup.group.id}
        group={sceneGroup.group}
        className="d-block"
      />
    </div>
  ));

  return (
    <HoverPopover
      placement="bottom"
      content={popoverContent}
      className="tag-tooltip"
    >
      <Button className="minimal">
        <Icon icon={faFilm} />
        <span>{scene.groups.length}</span>
      </Button>
    </HoverPopover>
  );
}

function maybeRenderSceneMarkerPopoverButton(scene: GQL.SlimSceneDataFragment) {
  if (scene.scene_markers.length <= 0) return;

  const popoverContent = scene.scene_markers.map((marker) => {
    const markerWithScene = { ...marker, scene: { id: scene.id } };
    return <SceneMarkerLink key={marker.id} marker={markerWithScene} />;
  });

  return (
    <HoverPopover placement="bottom" content={popoverContent}>
      <Button className="minimal">
        <Icon icon={faMapMarkerAlt} />
        <span>{scene.scene_markers.length}</span>
      </Button>
    </HoverPopover>
  );
}

function maybeRenderOCounter(scene: GQL.SlimSceneDataFragment) {
  if (scene.o_counter) {
    return (
      <div>
        <Button className="minimal">
          <span className="fa-icon">
            <SweatDrops />
          </span>
          <span>{scene.o_counter}</span>
        </Button>
      </div>
    );
  }
}

function maybeRenderGallery(scene: GQL.SlimSceneDataFragment) {
  if (scene.galleries.length <= 0) return;

  const popoverContent = scene.galleries.map((gallery) => (
    <GalleryLink key={gallery.id} gallery={gallery} />
  ));

  return (
    <HoverPopover placement="bottom" content={popoverContent}>
      <Button className="minimal">
        <Icon icon={faImages} />
        <span>{scene.galleries.length}</span>
      </Button>
    </HoverPopover>
  );
}

const FileCountPopoverButton: React.FC<{
  scene: GQL.SlimSceneDataFragment;
}> = ({ scene }) => {
  const intl = useIntl();

  if (scene.files.length <= 1) return null;

  const popoverContent = (
    <FormattedMessage
      id="files_amount"
      values={{ value: intl.formatNumber(scene.files.length ?? 0) }}
    />
  );

  return (
    <HoverPopover placement="bottom" content={popoverContent}>
      <Button className="minimal">
        <Icon icon={faFileAlt} />
        <span>{scene.files.length}</span>
      </Button>
    </HoverPopover>
  );
};

function maybeRenderOrganized(scene: GQL.SlimSceneDataFragment) {
  if (scene.organized) {
    return (
      <div>
        <Button className="minimal">
          <Icon icon={faBox} />
        </Button>
      </div>
    );
  }
}

function maybeRenderPopoverButtonGroup(scene: GQL.SlimSceneDataFragment) {
  if (
    scene.tags.length > 0 ||
    scene.performers.length > 0 ||
    scene.groups.length > 0 ||
    scene.scene_markers.length > 0 ||
    scene?.o_counter ||
    scene.galleries.length > 0 ||
    scene.files.length > 1 ||
    scene.organized
  ) {
    return (
      <ButtonGroup className="flex-wrap">
        {maybeRenderTagPopoverButton(scene)}
        {maybeRenderPerformerPopoverButton(scene)}
        {maybeRenderGroupPopoverButton(scene)}
        {maybeRenderSceneMarkerPopoverButton(scene)}
        {maybeRenderOCounter(scene)}
        {maybeRenderGallery(scene)}
        <FileCountPopoverButton scene={scene} />
        {maybeRenderOrganized(scene)}
      </ButtonGroup>
    );
  }
}

const SceneSprite: React.FC<{
  scene: GQL.SlimSceneDataFragment;
  screenshotFallback: boolean;
  style?: React.CSSProperties;
}> = ({ scene, screenshotFallback, style }) => {
  // the sprite path is always set, even if no sprite was generated
  const [spriteMissing, setSpriteMissing] = useState(false);

  const src =
    (spriteMissing ? scene.paths.screenshot : scene.paths.sprite) ?? "";

  function onError() {
    if (screenshotFallback && !spriteMissing) {
      setSpriteMissing(true);
    }
  }

  return (
    <HoverPopover
      content={<img src={src} alt="" width={600} />}
      placement="right"
    >
      <img src={src} alt="" width={100} style={style} onError={onError} />
    </HoverPopover>
  );
};

interface ISceneFileDetailsHeaderProps {
  // prefix for the column class names
  className?: string;
  operationsHeader?: React.ReactNode;
}

export const SceneFileDetailsHeader: React.FC<ISceneFileDetailsHeaderProps> = ({
  className = CLASSNAME,
  operationsHeader,
}) => {
  const intl = useIntl();

  return (
    <>
      <colgroup>
        <col className={`${className}-checkbox`} />
        <col className={`${className}-sprite`} />
        <col className={`${className}-title`} />
        <col className={`${className}-details`} />
        <col className={`${className}-duration`} />
        <col className={`${className}-filesize`} />
        <col className={`${className}-resolution`} />
        <col className={`${className}-bitrate`} />
        <col className={`${className}-codec`} />
        <col className={`${className}-mod-time`} />
        {operationsHeader !== undefined && (
          <col className={`${className}-operations`} />
        )}
      </colgroup>
      <thead>
        <tr>
          <th> </th>
          <th> </th>
          <th>{intl.formatMessage({ id: "details" })}</th>
          <th> </th>
          <th>{intl.formatMessage({ id: "duration" })}</th>
          <th>{intl.formatMessage({ id: "filesize" })}</th>
          <th>{intl.formatMessage({ id: "resolution" })}</th>
          <th>{intl.formatMessage({ id: "bitrate" })}</th>
          <th>{intl.formatMessage({ id: "media_info.video_codec" })}</th>
          <th>{intl.formatMessage({ id: "file_mod_time_short" })}</th>
          {operationsHeader !== undefined && <th>{operationsHeader}</th>}
        </tr>
      </thead>
    </>
  );
};

interface ISceneFileDetailsRowProps {
  scene: GQL.SlimSceneDataFragment;
  checked: boolean;
  onCheck: (checked: boolean, shiftKey: boolean) => void;
  sceneLink?: string;
  className?: string;
  // highlights checked scenes as marked for deletion
  markChecked?: boolean;
  screenshotFallback?: boolean;
  operations?: React.ReactNode;
}

export const SceneFileDetailsRow: React.FC<ISceneFileDetailsRowProps> = ({
  scene,
  checked,
  onCheck,
  sceneLink,
  className,
  markChecked = false,
  screenshotFallback = false,
  operations,
}) => {
  const file = scene.files.length > 0 ? scene.files[0] : undefined;
  const marked = markChecked && checked;

  let shiftKey = false;

  return (
    <tr className={className}>
      <td>
        <Form.Check
          checked={checked}
          onChange={(e) => onCheck(e.currentTarget.checked, shiftKey)}
          onClick={(event: React.MouseEvent<HTMLInputElement, MouseEvent>) => {
            shiftKey = event.shiftKey;
            event.stopPropagation();
          }}
        />
      </td>
      <td>
        <SceneSprite
          scene={scene}
          screenshotFallback={screenshotFallback}
          style={{ border: marked ? "2px solid red" : "" }}
        />
      </td>
      <td className="text-left">
        <p>
          <Link
            to={sceneLink ?? `/scenes/${scene.id}`}
            style={{
              fontWeight: marked ? "bold" : "inherit",
              textDecoration: marked ? "line-through 3px" : "inherit",
              textDecorationColor: marked ? "red" : "inherit",
            }}
          >
            {" "}
            {scene.title
              ? scene.title
              : TextUtils.fileNameFromPath(file?.path ?? "")}{" "}
          </Link>
        </p>
        <p className="scene-path">{file?.path ?? ""}</p>
      </td>
      <td className="scene-details">{maybeRenderPopoverButtonGroup(scene)}</td>
      <td>{file?.duration && TextUtils.secondsToTimestamp(file.duration)}</td>
      <td>
        <FileSize size={file?.size ?? 0} />
      </td>
      <td>{`${file?.width ?? 0}x${file?.height ?? 0}`}</td>
      <td>
        <FormattedNumber
          value={(file?.bit_rate ?? 0) / 1000000}
          maximumFractionDigits={2}
        />
        &nbsp;mbps
      </td>
      <td>{file?.video_codec ?? ""}</td>
      <td>
        {file ? (
          <FormattedTime
            dateStyle="medium"
            timeStyle="medium"
            value={file.mod_time}
          />
        ) : (
          ""
        )}
      </td>
      {operations !== undefined && <td>{operations}</td>}
    </tr>
  );
};

interface ISceneFileDetailsTableProps {
  scenes: GQL.SlimSceneDataFragment[];
  queue?: SceneQueue;
  selectedIds: Set<string>;
  onSelectChange: (id: string, selected: boolean, shiftKey: boolean) => void;
}

export const SceneFileDetailsTable: React.FC<ISceneFileDetailsTableProps> = ({
  scenes,
  queue,
  selectedIds,
  onSelectChange,
}) => {
  return (
    <Table responsive striped className={`${CLASSNAME}-table`}>
      <SceneFileDetailsHeader />
      <tbody>
        {scenes.map((scene, index) => (
          <SceneFileDetailsRow
            key={scene.id}
            scene={scene}
            checked={selectedIds.has(scene.id)}
            onCheck={(checked, shiftKey) =>
              onSelectChange(scene.id, checked, shiftKey)
            }
            sceneLink={queue?.makeLink(scene.id, { sceneIndex: index })}
            screenshotFallback
          />
        ))}
      </tbody>
    </Table>
  );
};
