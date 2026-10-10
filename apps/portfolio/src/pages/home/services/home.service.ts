import { parseMarkdown } from '@/shared/lib/markdown.util';

import homeMarkdown from '../content/index.md?raw';

export interface HeroData {
  name: string;
  title: string;
  subtitle: string;
  description: string;
}

type HeroMetadata = Omit<HeroData, 'description'>;

const homeDocument = parseMarkdown<HeroMetadata>(homeMarkdown);

export const HERO: HeroData = { ...homeDocument.attributes, description: homeDocument.html };
