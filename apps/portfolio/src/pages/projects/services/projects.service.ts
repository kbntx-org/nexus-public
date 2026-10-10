import { parseMarkdown } from '@/shared/lib/markdown.util';

import nexusKubernetesMarkdown from '../content/nexus-kubernetes.md?raw';
import nexusObservabilityMarkdown from '../content/nexus-observability.md?raw';
import portfolioMarkdown from '../content/portfolio.md?raw';

export interface Project {
  title: string;
  description: string;
  tech: string[];
  features: string[];
  content: string;
  liveUrl?: string;
  githubUrl?: string;
  images?: string[];
  logo?: string;
}

type ProjectMetadata = Omit<Project, 'content'>;

export const PROJECTS: Project[] = [
  nexusObservabilityMarkdown,
  nexusKubernetesMarkdown,
  portfolioMarkdown
]
  .map(raw => parseMarkdown<ProjectMetadata>(raw))
  .map(document => ({ ...document.attributes, content: document.html }));
