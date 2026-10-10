import { marked } from 'marked';
import { parse as parseYaml } from 'yaml';

const FRONT_MATTER_PATTERN = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/;

export interface MarkdownDocument<Attributes> {
  attributes: Attributes;
  html: string;
}

export function parseMarkdown<Attributes>(raw: string): MarkdownDocument<Attributes> {
  const [, frontMatter = '', body = raw] = raw.match(FRONT_MATTER_PATTERN) ?? [];
  return {
    attributes: parseYaml(frontMatter) as Attributes,
    html: marked.parse(body, { async: false })
  };
}
