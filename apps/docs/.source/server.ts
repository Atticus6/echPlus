// @ts-nocheck
import * as __fd_glob_12 from "../content/docs/server/index.mdx?collection=docs"
import * as __fd_glob_11 from "../content/docs/server/docker.mdx?collection=docs"
import * as __fd_glob_10 from "../content/docs/server/cloudflare.mdx?collection=docs"
import * as __fd_glob_9 from "../content/docs/server/cli.mdx?collection=docs"
import * as __fd_glob_8 from "../content/docs/client/index.mdx?collection=docs"
import * as __fd_glob_7 from "../content/docs/client/docker.mdx?collection=docs"
import * as __fd_glob_6 from "../content/docs/client/cli.mdx?collection=docs"
import * as __fd_glob_5 from "../content/docs/principle.mdx?collection=docs"
import * as __fd_glob_4 from "../content/docs/index.mdx?collection=docs"
import * as __fd_glob_3 from "../content/docs/desktop.mdx?collection=docs"
import { default as __fd_glob_2 } from "../content/docs/server/meta.json?collection=docs"
import { default as __fd_glob_1 } from "../content/docs/client/meta.json?collection=docs"
import { default as __fd_glob_0 } from "../content/docs/meta.json?collection=docs"
import { server } from 'fumadocs-mdx/runtime/server';
import type * as Config from '../source.config';

const create = server<typeof Config, import("fumadocs-mdx/runtime/types").InternalTypeConfig & {
  DocData: {
  }
}>({"doc":{"passthroughs":["extractedReferences"]}});

export const docs = await create.docs("docs", "content/docs", {"meta.json": __fd_glob_0, "client/meta.json": __fd_glob_1, "server/meta.json": __fd_glob_2, }, {"desktop.mdx": __fd_glob_3, "index.mdx": __fd_glob_4, "principle.mdx": __fd_glob_5, "client/cli.mdx": __fd_glob_6, "client/docker.mdx": __fd_glob_7, "client/index.mdx": __fd_glob_8, "server/cli.mdx": __fd_glob_9, "server/cloudflare.mdx": __fd_glob_10, "server/docker.mdx": __fd_glob_11, "server/index.mdx": __fd_glob_12, });