import React from "react";
import { Button } from "react-bootstrap";
import { FormattedMessage } from "react-intl";
import { Link } from "react-router-dom";
import { Setting } from "./Inputs";
import { SettingSection } from "./SettingSection";
import { SettingsTombstonesSection } from "./SettingsTombstonesSection";
import { PatchContainerComponent } from "src/patch";
import { ExternalLink } from "../Shared/ExternalLink";

const SettingsToolsSection = PatchContainerComponent("SettingsToolsSection");

export const SettingsToolsPanel: React.FC = () => {
  return (
    <>
      <SettingSection headingID="config.tools.heading">
        <SettingsToolsSection>
          <Setting
            heading={
              <ExternalLink href="/playground">
                <Button>
                  <FormattedMessage id="config.tools.graphql_playground" />
                </Button>
              </ExternalLink>
            }
          />
        </SettingsToolsSection>
      </SettingSection>
      <SettingSection headingID="config.tools.scene_tools">
        <SettingsToolsSection>
          <Setting
            heading={
              <Link to="/sceneFilenameParser">
                <Button>
                  <FormattedMessage id="config.tools.scene_filename_parser.title" />
                </Button>
              </Link>
            }
          />

          <Setting
            heading={
              <Link to="/sceneDuplicateChecker">
                <Button>
                  <FormattedMessage id="config.tools.scene_duplicate_checker" />
                </Button>
              </Link>
            }
          />

          <Setting
            heading={
              <Link to="/tombstones">
                <Button>
                  <FormattedMessage id="config.tools.tombstones" />
                </Button>
              </Link>
            }
          />
        </SettingsToolsSection>
      </SettingSection>
      <SettingsTombstonesSection />
    </>
  );
};
