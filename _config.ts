import lume from "lume/mod.ts";
import base_path from "lume/plugins/base_path.ts";
import date from "lume/plugins/date.ts";
import favicon from "lume/plugins/favicon.ts";
import feed from "lume/plugins/feed.ts";

const site = lume();

site.use(base_path());
site.use(date());
site.use(favicon());
site.use(feed());

export default site;
