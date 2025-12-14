// @ts-nocheck
import { browser } from 'fumadocs-mdx/runtime/browser';
import type * as Config from '../source.config';

const create = browser<typeof Config, import("fumadocs-mdx/runtime/types").InternalTypeConfig & {
  DocData: {
  }
}>();
const browserCollections = {
  docs: create.doc("docs", {"desktop.mdx": () => import("../content/docs/desktop.mdx?collection=docs"), "index.mdx": () => import("../content/docs/index.mdx?collection=docs"), "principle.mdx": () => import("../content/docs/principle.mdx?collection=docs"), "server/cli.mdx": () => import("../content/docs/server/cli.mdx?collection=docs"), "server/cloudflare.mdx": () => import("../content/docs/server/cloudflare.mdx?collection=docs"), "server/docker.mdx": () => import("../content/docs/server/docker.mdx?collection=docs"), "server/index.mdx": () => import("../content/docs/server/index.mdx?collection=docs"), "client/cli.mdx": () => import("../content/docs/client/cli.mdx?collection=docs"), "client/docker.mdx": () => import("../content/docs/client/docker.mdx?collection=docs"), "client/index.mdx": () => import("../content/docs/client/index.mdx?collection=docs"), }),
};
export default browserCollections;