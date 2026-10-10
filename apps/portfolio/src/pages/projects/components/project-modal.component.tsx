import { ChevronLeft, ChevronRight, ExternalLink, Github, X } from 'lucide-react';
import { useEffect, useState } from 'react';

import type { Project } from '../services/projects.service';

const MOBILE_BREAKPOINT_PX = 1024;
const OPENING_DELAY_MS = 10;

const ICON_BUTTON_CLASS =
  'flex h-8 w-8 items-center justify-center rounded-full text-night-text-muted transition hover:bg-white/10 hover:text-night-text';
const CAROUSEL_ARROW_CLASS =
  'absolute top-1/2 z-10 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-night-sky/60 text-white/80 backdrop-blur-sm transition hover:bg-night-sky/80 hover:text-white desktop:h-9 desktop:w-9';
const SECTION_TITLE_CLASS =
  'mb-2.5 text-xs font-semibold uppercase tracking-wider text-night-text-muted desktop:mb-3 desktop:text-sm';

interface ProjectModalProps {
  project: Project;
  onClose: () => void;
}

export function ProjectModal({ project, onClose }: ProjectModalProps) {
  const images = project.images ?? [];
  const [currentImageIndex, setCurrentImageIndex] = useState(0);
  const [isOpening, setIsOpening] = useState(false);
  const [isMobileViewport] = useState(() => window.innerWidth < MOBILE_BREAKPOINT_PX);

  useEffect(() => {
    document.body.style.overflow = 'hidden';
    const openingTimeout = setTimeout(() => setIsOpening(true), OPENING_DELAY_MS);
    return () => {
      clearTimeout(openingTimeout);
      document.body.style.overflow = '';
    };
  }, []);

  const nextImage = () => setCurrentImageIndex(index => (index + 1) % images.length);
  const previousImage = () =>
    setCurrentImageIndex(index => (index === 0 ? images.length - 1 : index - 1));

  const slideClass = isOpening ? 'translate-y-0' : isMobileViewport ? 'translate-y-full' : '';

  return (
    <div
      className="glass-scrim fixed inset-0 z-50 flex items-end justify-center desktop:items-center"
      onClick={onClose}
    >
      <section
        className={`glass-card relative box-border flex max-h-[90svh] w-full flex-col overflow-hidden ring-1 ring-white/5 transition-transform duration-300 ease-out mobile:rounded-b-none desktop:mx-4 desktop:max-h-[85vh] desktop:max-w-3xl ${slideClass}`}
        onClick={event => event.stopPropagation()}
      >
        <header className="flex flex-shrink-0 items-center justify-between px-5 py-4 desktop:px-8 desktop:py-5">
          <h2 className="text-xl font-bold text-night-text desktop:text-2xl">{project.title}</h2>
          <div className="flex items-center gap-1">
            {project.githubUrl && (
              <a
                href={project.githubUrl}
                target="_blank"
                rel="noopener"
                className={ICON_BUTTON_CLASS}
              >
                <Github className="h-4 w-4" />
              </a>
            )}
            <button className={ICON_BUTTON_CLASS} onClick={onClose}>
              <X className="h-5 w-5" />
            </button>
          </div>
        </header>

        <div className="overflow-y-auto px-5 pb-5 desktop:px-8 desktop:pb-8">
          {images.length > 0 && (
            <div className="mb-5 overflow-hidden rounded-xl border border-white/5 desktop:mb-8">
              <div className="relative h-52 w-full overflow-hidden bg-night-sky/30 desktop:h-80">
                <div
                  className="flex h-full w-full transition-transform duration-500 ease-out"
                  style={{ transform: `translateX(-${currentImageIndex * 100}%)` }}
                >
                  {images.map((image, index) => (
                    <div key={image} className="h-full w-full flex-shrink-0">
                      <img
                        src={image}
                        alt={`${project.title} image ${index + 1}`}
                        className="h-full w-full object-contain p-3 desktop:p-4"
                      />
                    </div>
                  ))}
                </div>

                {images.length > 1 && (
                  <>
                    <button
                      className={`${CAROUSEL_ARROW_CLASS} left-2 desktop:left-3`}
                      onClick={previousImage}
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </button>
                    <button
                      className={`${CAROUSEL_ARROW_CLASS} right-2 desktop:right-3`}
                      onClick={nextImage}
                    >
                      <ChevronRight className="h-4 w-4" />
                    </button>
                    <div className="absolute bottom-2 left-1/2 z-10 flex -translate-x-1/2 gap-1.5 rounded-full bg-night-sky/50 px-2 py-1 backdrop-blur-sm desktop:bottom-3 desktop:px-2.5 desktop:py-1.5">
                      {images.map((image, index) => (
                        <button
                          key={image}
                          className={`h-1.5 rounded-full transition-all duration-200 ${
                            index === currentImageIndex
                              ? 'w-3.5 bg-night-gold desktop:w-4'
                              : 'w-1.5 bg-white/30 hover:bg-white/50'
                          }`}
                          onClick={() => setCurrentImageIndex(index)}
                        ></button>
                      ))}
                    </div>
                  </>
                )}
              </div>
            </div>
          )}

          <div className="space-y-5 desktop:space-y-6">
            <div
              className="text-sm leading-relaxed text-night-text-soft desktop:text-base"
              dangerouslySetInnerHTML={{ __html: project.content }}
            ></div>

            {project.features && project.features.length > 0 && (
              <div>
                <h3 className={SECTION_TITLE_CLASS}>Features</h3>
                <div className="grid grid-cols-1 gap-2 desktop:grid-cols-2">
                  {project.features.map(feature => (
                    <div key={feature} className="flex items-start gap-2.5">
                      <span className="mt-0.5 text-night-gold">✓</span>
                      <span className="text-sm text-night-text-soft">{feature}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div>
              <h3 className={SECTION_TITLE_CLASS}>Stack</h3>
              <div className="flex flex-wrap gap-1.5">
                {project.tech.map(tech => (
                  <span
                    key={tech}
                    className="rounded-full bg-white/5 px-3 py-1 text-xs font-medium text-night-text-muted"
                  >
                    {tech}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </div>

        {project.liveUrl && (
          <footer className="flex flex-shrink-0 flex-wrap gap-3 border-t border-white/10 bg-white/5 px-5 py-4 backdrop-blur-sm desktop:px-8 desktop:py-5">
            <a
              href={project.liveUrl}
              target="_blank"
              rel="noopener"
              className="gold-gradient inline-flex flex-1 items-center justify-center gap-2 rounded-full px-5 py-2.5 text-sm font-medium text-night-sky transition hover:brightness-110 active:scale-95 desktop:flex-initial"
            >
              <ExternalLink className="h-4 w-4" />
              Live Demo
            </a>
          </footer>
        )}
      </section>
    </div>
  );
}
