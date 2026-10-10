import { useState } from 'react';

import { ProjectModal } from './components/project-modal.component';
import { PROJECTS, type Project } from './services/projects.service';

const VISIBLE_TECH_COUNT = 6;
const LOGO_GRADIENTS = [
  'bg-gradient-to-br from-blue-500 to-purple-600',
  'bg-gradient-to-br from-green-500 to-teal-600',
  'bg-gradient-to-br from-purple-500 to-pink-600',
  'bg-gradient-to-br from-orange-500 to-red-600',
  'bg-gradient-to-br from-pink-500 to-rose-600',
  'bg-gradient-to-br from-cyan-500 to-blue-600'
];

export function ProjectsPage() {
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);

  return (
    <>
      <div className="relative z-10 py-8 md:py-12">
        <div className="mx-auto max-w-6xl px-4">
          <header
            className="animate-slide-in-up opacity-0"
            style={{ animationDelay: '0.1s', animationFillMode: 'forwards' }}
          >
            <h1 className="mb-4 text-center text-4xl font-bold text-night-text sm:text-5xl lg:text-6xl">
              My Projects
            </h1>
            <p className="mb-12 text-center text-lg text-night-text-muted sm:text-xl">
              A showcase of my recent work and technical expertise
            </p>
          </header>

          <section
            className="grid grid-cols-1 gap-6 md:grid-cols-2 md:gap-8 lg:grid-cols-3"
            aria-label="Projects"
          >
            {PROJECTS.map((project, index) => (
              <article
                key={project.title}
                className="glass-card-sm flex animate-slide-in-up flex-col overflow-hidden opacity-0 transition hover:border-white/20 hover:shadow-xl"
                style={{ animationDelay: '0.5s', animationFillMode: 'forwards' }}
              >
                <div className="flex flex-1 flex-col p-6">
                  <header className="mb-3 flex items-center gap-3">
                    {project.logo ? (
                      <img
                        src={project.logo}
                        alt={`${project.title} logo`}
                        className="h-12 w-12 flex-shrink-0 object-contain"
                      />
                    ) : (
                      <div
                        className={`flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full text-lg font-semibold text-white shadow-lg ${LOGO_GRADIENTS[index % LOGO_GRADIENTS.length]}`}
                      >
                        {project.title.charAt(0)}
                      </div>
                    )}
                    <h3 className="text-xl font-semibold text-night-text">{project.title}</h3>
                  </header>

                  <p className="mb-4 leading-relaxed text-night-text-muted">
                    {project.description}
                  </p>

                  <div className="mb-6 flex flex-wrap gap-2">
                    {project.tech.slice(0, VISIBLE_TECH_COUNT).map(tech => (
                      <span
                        key={tech}
                        className="rounded-full bg-white/10 px-3 py-1 text-sm font-medium text-night-text-soft"
                      >
                        {tech}
                      </span>
                    ))}
                  </div>

                  <div className="mt-auto flex justify-start gap-3">
                    <button
                      className="rounded-md border border-white/15 bg-white/5 px-4 py-2 font-medium text-night-text transition hover:bg-white/10"
                      onClick={() => setSelectedProject(project)}
                    >
                      View Project
                    </button>
                    {project.githubUrl && (
                      <a
                        href={project.githubUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="rounded-md border border-white/10 bg-transparent px-4 py-2 font-medium text-night-text-muted transition hover:bg-white/10 hover:text-night-text"
                      >
                        Source Code
                      </a>
                    )}
                  </div>
                </div>
              </article>
            ))}
          </section>
        </div>
      </div>

      {selectedProject && (
        <ProjectModal project={selectedProject} onClose={() => setSelectedProject(null)} />
      )}
    </>
  );
}
