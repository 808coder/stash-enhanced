import * as GQL from "src/core/generated-graphql";

export type TombstoneData = GQL.TombstoneDataFragment;
export type TombstonePerformer = TombstoneData["performers"][number];
export type TombstoneFile = TombstoneData["files"][number];
