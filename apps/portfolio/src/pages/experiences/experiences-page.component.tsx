import { ChevronDown } from 'lucide-react';
import { useState } from 'react';

import { EXPERIENCES, type Role } from './data/experiences.data';

const VISIBLE_TECH_COUNT = 6;
const CARD_BASE_DELAY_SECONDS = 0.5;
const CARD_DELAY_STEP_SECONDS = 0.1;

function RoleDetails({ role }: { role: Role }) {
  const [roleExpanded, setRoleExpanded] = useState(false);
  const [techExpanded, setTechExpanded] = useState(false);
  const visibleTechnologies = techExpanded
    ? role.technologies
    : role.technologies.slice(0, VISIBLE_TECH_COUNT);

  return (
    <div className="border-l-4 border-night-purple pl-6">
      <div className="mb-4 flex flex-col md:flex-row md:items-center md:justify-between">
        <div className="flex-1">
          <h3 className="text-xl font-semibold">{role.title}</h3>
          <span className="mt-1 block text-sm text-night-text-muted">{role.duration}</span>
        </div>
        <button
          onClick={() => setRoleExpanded(expanded => !expanded)}
          className="mt-2 flex items-center gap-2 text-night-purple transition-colors duration-200 md:mt-0"
        >
          <span className="text-sm font-medium">{roleExpanded ? 'Show Less' : 'Show Details'}</span>
          <ChevronDown
            className={`h-4 w-4 transition-transform duration-200 ${roleExpanded ? 'rotate-180' : ''}`}
          />
        </button>
      </div>

      {roleExpanded && (
        <div className="space-y-6 pb-4">
          <p className="leading-relaxed text-night-text-soft">{role.description}</p>

          {role.highlights && role.highlights.length > 0 && (
            <ul className="space-y-2">
              {role.highlights.map(highlight => (
                <li key={highlight} className="flex items-start gap-3 text-night-text-muted">
                  <span className="mt-2 h-2 w-2 flex-shrink-0 rounded-full bg-night-purple"></span>
                  <span>{highlight}</span>
                </li>
              ))}
            </ul>
          )}

          <section>
            <h4 className="mb-3 text-lg font-medium">Technologies</h4>
            <div className="flex flex-wrap items-center gap-2">
              {visibleTechnologies.map(tech => (
                <span
                  key={tech}
                  className="rounded-full bg-white/10 px-3 py-1 text-sm font-medium text-night-text-soft"
                >
                  {tech}
                </span>
              ))}
              {role.technologies.length > VISIBLE_TECH_COUNT && (
                <button
                  onClick={() => setTechExpanded(expanded => !expanded)}
                  className="rounded-full border border-night-purple px-3 py-1 text-sm font-medium text-night-purple transition-colors hover:bg-night-purple/10"
                >
                  {techExpanded
                    ? 'Show less'
                    : `+${role.technologies.length - VISIBLE_TECH_COUNT} more`}
                </button>
              )}
            </div>
          </section>
        </div>
      )}
    </div>
  );
}

export function ExperiencesPage() {
  return (
    <div className="relative z-10 py-8 text-night-text md:py-12">
      <div className="mx-auto max-w-6xl px-4">
        <header
          className="animate-slide-in-up opacity-0"
          style={{ animationDelay: '0.1s', animationFillMode: 'forwards' }}
        >
          <h1 className="mb-4 text-center text-4xl font-bold sm:text-5xl lg:text-6xl">
            My Experiences
          </h1>
          <p className="mb-12 text-center text-lg text-night-text-muted sm:text-xl">
            My professional journey and career progression
          </p>
        </header>

        <div className="space-y-12">
          {EXPERIENCES.map((experience, experienceIndex) => (
            <article
              key={experience.company}
              className="glass-card-sm animate-slide-in-up p-6 opacity-0 md:p-8"
              style={{
                animationDelay: `${CARD_BASE_DELAY_SECONDS + experienceIndex * CARD_DELAY_STEP_SECONDS}s`,
                animationFillMode: 'forwards'
              }}
            >
              <header className="mb-4 flex flex-col md:flex-row md:items-center md:justify-between">
                <div>
                  <h2 className="mb-2 text-2xl font-bold md:text-3xl">{experience.company}</h2>
                  <p className="text-night-text-muted">{experience.location}</p>
                </div>
                <span className="mt-4 self-start rounded-full bg-gradient-to-r from-night-purple to-night-purple-deep px-4 py-2 text-sm font-medium text-white md:mt-0">
                  {experience.duration}
                </span>
              </header>

              <p className="mb-6 leading-relaxed text-night-text-soft">{experience.description}</p>

              <div className="space-y-4">
                {experience.roles.map(role => (
                  <RoleDetails key={role.title} role={role} />
                ))}
              </div>
            </article>
          ))}
        </div>
      </div>
    </div>
  );
}
