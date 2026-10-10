import { Github, Linkedin } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

import { GITHUB_URL, LINKEDIN_URL } from '@/shared/lib/social-links.constants';

import { SkillsSection } from './components/skills-section.component';
import { HERO } from './services/home.service';

const CV_URL = 'https://portfolio-assets.kbntx.com/cv-kenny-talbi.pdf';

export function HomePage() {
  const navigate = useNavigate();

  return (
    <div className="font-roboto">
      <section className="relative z-10 min-h-dvh overflow-hidden sm:min-h-[calc(100dvh-4rem)]">
        <div className="relative mx-auto flex min-h-dvh max-w-6xl items-center px-4 py-16 sm:min-h-[calc(100dvh-4rem)]">
          <div className="max-w-2xl animate-slide-in-up">
            <div className="mb-6 flex items-center gap-4">
              <div className="h-16 w-16 shrink-0 overflow-hidden rounded-full border-4 border-transparent bg-gradient-to-br from-night-purple to-night-purple-deep bg-clip-border shadow-lg shadow-purple-500/20">
                <img
                  src="/assets/images/kenny.webp"
                  alt="Kenny"
                  className="h-full w-full object-cover"
                />
              </div>
              <h1 className="text-4xl font-bold leading-tight text-night-text sm:text-5xl lg:text-6xl">
                Hi, I'm{' '}
                <span className="relative bg-gradient-to-r from-night-purple to-night-purple-deep bg-clip-text text-transparent">
                  {HERO.name}
                  <div className="absolute -bottom-2 left-0 h-1 w-full scale-x-0 transform animate-expand-width bg-gradient-to-r from-night-purple to-night-purple-deep"></div>
                </span>
              </h1>
            </div>
            <p className="mb-4 text-sm text-night-text-muted">{HERO.title}</p>
            <p className="mb-8 text-lg leading-relaxed text-night-text-soft sm:text-xl">
              {HERO.subtitle}
            </p>
            <div className="flex flex-wrap items-center gap-4">
              <button
                className="group relative inline-flex items-center gap-2 overflow-hidden rounded-lg bg-gradient-to-r from-night-gold to-night-gold-deep px-6 py-3 font-medium text-night-sky shadow-lg shadow-amber-500/20 transition-all duration-300 hover:shadow-xl hover:shadow-amber-500/30"
                onClick={() => navigate('/experiences')}
              >
                <span className="relative z-10">View Experiences</span>
                <div className="absolute inset-0 -translate-x-full transform bg-gradient-to-r from-night-gold-deep to-night-gold transition-transform duration-500 group-hover:translate-x-0"></div>
              </button>
              <a
                href={CV_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="group relative inline-flex items-center gap-2 overflow-hidden rounded-lg border border-white/15 bg-white/5 px-6 py-3 font-medium text-night-text shadow-lg backdrop-blur-sm transition-all duration-300 hover:bg-white/10 hover:shadow-xl"
              >
                <span className="relative z-10">Open CV</span>
                <div className="absolute inset-0 -translate-x-full transform bg-white/5 transition-transform duration-500 group-hover:translate-x-0"></div>
              </a>
              <a
                href={LINKEDIN_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="hidden h-10 w-10 shrink-0 items-center justify-center rounded-full border border-white/10 bg-[#0077b5]/80 text-white shadow-lg backdrop-blur-sm transition-all duration-300 hover:bg-[#0077b5] hover:shadow-xl sm:flex"
              >
                <Linkedin size={16} />
              </a>
              <a
                href={GITHUB_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="hidden h-10 w-10 shrink-0 items-center justify-center rounded-full border border-white/10 bg-white/10 text-white shadow-lg backdrop-blur-sm transition-all duration-300 hover:bg-white/20 hover:shadow-xl sm:flex"
              >
                <Github size={16} />
              </a>
            </div>
          </div>
        </div>
      </section>

      <section className="relative z-10 py-12 md:py-8">
        <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-night-purple/30 to-transparent"></div>
        <div className="mx-auto max-w-6xl px-4">
          <div className="animate-slide-in-up">
            <h2 className="relative mb-12 text-center text-3xl font-bold text-night-text sm:text-4xl">
              About Me
              <div className="absolute -bottom-3 left-1/2 h-1 w-16 -translate-x-1/2 transform rounded-full bg-gradient-to-r from-night-purple to-night-purple-deep"></div>
            </h2>
            <div className="mx-auto max-w-4xl">
              <div className="rounded-xl border border-white/10 bg-night-card/40 p-8 text-lg leading-relaxed text-night-text-soft shadow-xl backdrop-blur-sm">
                <div dangerouslySetInnerHTML={{ __html: HERO.description }}></div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="relative z-10">
        <SkillsSection />
      </div>
    </div>
  );
}
