import React from "react";
import * as GQL from "src/core/generated-graphql";
import TextUtils from "src/utils/text";
import { GalleryLink, TagLink } from "src/components/Shared/TagLink";
import { DetailPanelPerformers } from "src/components/Shared/DetailPanelPerformers";
import { FormattedMessage, useIntl } from "react-intl";
import { PhotographerLink } from "src/components/Shared/Link";
import { PatchComponent } from "../../../patch";
import { CustomFields } from "src/components/Shared/CustomFields";
import { CollapsibleDetails } from "src/components/Shared/CollapsibleDetails";
interface IImageDetailProps {
  image: GQL.ImageDataFragment;
}

export const ImageDetailPanel: React.FC<IImageDetailProps> = PatchComponent(
  "ImageDetailPanel",
  (props) => {
    const intl = useIntl();

    function renderDetails() {
      if (!props.image.details) return;
      return (
        <CollapsibleDetails
          key={`details-${props.image.id}`}
          details={props.image.details}
        />
      );
    }

    function renderTags() {
      if (props.image.tags.length === 0) return;
      const tags = props.image.tags.map((tag) => (
        <TagLink key={tag.id} tag={tag} linkType="image" />
      ));
      return (
        <>
          <h6>
            <FormattedMessage
              id="countables.tags"
              values={{ count: props.image.tags.length }}
            />
          </h6>
          {tags}
        </>
      );
    }

    function renderPerformers() {
      if (props.image.performers.length === 0) return;
      return (
        <DetailPanelPerformers
          key={`performers-${props.image.id}`}
          performers={props.image.performers}
          ageFromDate={props.image.date ?? undefined}
          className="image-performers"
        />
      );
    }

    function renderGalleries() {
      if (props.image.galleries.length === 0) return;
      const galleries = props.image.galleries.map((gallery) => (
        <GalleryLink key={gallery.id} gallery={gallery} />
      ));
      return (
        <>
          <h6>
            <FormattedMessage
              id="countables.galleries"
              values={{ count: props.image.galleries.length }}
            />
          </h6>
          {galleries}
        </>
      );
    }

    // filename should use entire row if there is no studio
    const imageDetailsWidth = props.image.studio ? "col-9" : "col-12";

    return (
      <>
        <div className="row">
          <div className={`${imageDetailsWidth} col-12 image-details`}>
            {renderGalleries()}
            {
              <h6>
                {" "}
                <FormattedMessage id="created_at" />:{" "}
                {TextUtils.formatDateTime(intl, props.image.created_at)}{" "}
              </h6>
            }
            {
              <h6>
                <FormattedMessage id="updated_at" />:{" "}
                {TextUtils.formatDateTime(intl, props.image.updated_at)}{" "}
              </h6>
            }
            {props.image.code && (
              <h6>
                <FormattedMessage id="scene_code" />: {props.image.code}{" "}
              </h6>
            )}
            {props.image.photographer && (
              <h6>
                <FormattedMessage id="photographer" />:{" "}
                <PhotographerLink
                  photographer={props.image.photographer}
                  linkType="image"
                />
              </h6>
            )}
          </div>
        </div>
        <div className="row">
          <div className="col-12">
            {renderDetails()}
            {renderTags()}
            {renderPerformers()}
            <CustomFields values={props.image.custom_fields} fullWidth />
          </div>
        </div>
      </>
    );
  }
);
