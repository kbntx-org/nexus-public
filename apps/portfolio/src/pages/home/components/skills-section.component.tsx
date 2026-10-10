import { useEffect, useRef } from 'react';

import { SKILLS, type Skill } from '../data/skills.data';

const ICONS_BASE_URL = 'https://portfolio-assets.kbntx.com/icons';
const MOBILE_BREAKPOINT_PX = 768;
const MOBILE_PARTICLE_SIZE_PX = 40;
const DESKTOP_PARTICLE_SIZE_PX = 48;
const MAXIMUM_SPEED_PX_PER_FRAME = 1.5;
const VISIBILITY_THRESHOLD = 0.1;
const VISIBILITY_ROOT_MARGIN = '0px 0px -50px 0px';
const ANIMATION_DELAY_STEP_SECONDS = 0.5;
const ANIMATION_BASE_DURATION_SECONDS = 8;
const ANIMATION_DURATION_STEP_SECONDS = 2;

interface Particle {
  x: number;
  y: number;
  velocityX: number;
  velocityY: number;
  element: HTMLElement;
}

function getIconUrl(skill: Skill): string {
  return `${ICONS_BASE_URL}/${skill.icon}`;
}

function getParticleSize(): number {
  return window.innerWidth < MOBILE_BREAKPOINT_PX
    ? MOBILE_PARTICLE_SIZE_PX
    : DESKTOP_PARTICLE_SIZE_PX;
}

function randomVelocity(): number {
  return (Math.random() - 0.5) * MAXIMUM_SPEED_PX_PER_FRAME;
}

function placeParticles(container: HTMLElement): Particle[] {
  const containerRect = container.getBoundingClientRect();
  const particleSize = getParticleSize();

  return Array.from(container.children as HTMLCollectionOf<HTMLElement>).map(element => {
    const x = Math.random() * (containerRect.width - particleSize);
    const y = Math.random() * (containerRect.height - particleSize);
    element.style.left = `${x}px`;
    element.style.top = `${y}px`;
    return { x, y, velocityX: randomVelocity(), velocityY: randomVelocity(), element };
  });
}

function moveParticles(container: HTMLElement, particles: Particle[]): void {
  const containerRect = container.getBoundingClientRect();
  const particleSize = getParticleSize();
  const maximumX = containerRect.width - particleSize;
  const maximumY = containerRect.height - particleSize;

  particles.forEach(particle => {
    particle.x += particle.velocityX;
    particle.y += particle.velocityY;

    if (particle.x <= 0 || particle.x >= maximumX) {
      particle.velocityX = -particle.velocityX;
      particle.x = Math.max(0, Math.min(particle.x, maximumX));
    }

    if (particle.y <= 0 || particle.y >= maximumY) {
      particle.velocityY = -particle.velocityY;
      particle.y = Math.max(0, Math.min(particle.y, maximumY));
    }

    particle.element.style.left = `${particle.x}px`;
    particle.element.style.top = `${particle.y}px`;
  });
}

export function SkillsSection() {
  const section = useRef<HTMLElement>(null);
  const particlesContainer = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const sectionElement = section.current;
    const container = particlesContainer.current;
    if (!sectionElement || !container) {
      return;
    }

    const particles = placeParticles(container);
    let animationFrameId: number | null = null;

    const animate = () => {
      moveParticles(container, particles);
      animationFrameId = requestAnimationFrame(animate);
    };

    const stop = () => {
      if (animationFrameId !== null) {
        cancelAnimationFrame(animationFrameId);
        animationFrameId = null;
      }
    };

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry.isIntersecting) {
          stop();
        } else if (animationFrameId === null) {
          animate();
        }
      },
      { threshold: VISIBILITY_THRESHOLD, rootMargin: VISIBILITY_ROOT_MARGIN }
    );
    observer.observe(sectionElement);

    return () => {
      observer.disconnect();
      stop();
    };
  }, []);

  return (
    <section ref={section} className="relative py-12 md:py-8">
      <div className="mx-auto max-w-6xl px-4">
        <h2 className="relative mb-12 animate-slide-in-up text-center text-3xl font-bold text-foreground sm:text-4xl">
          Skills & Technologies
          <div className="absolute -bottom-3 left-1/2 h-1 w-16 -translate-x-1/2 transform rounded-full bg-gradient-to-r from-night-purple to-night-purple-deep"></div>
        </h2>

        <div
          ref={particlesContainer}
          className="relative h-[300px] w-full max-w-full animate-slide-in-up overflow-visible md:h-[400px]"
          style={{ animationDelay: '0.2s' }}
        >
          {SKILLS.map((skill, index) => (
            <div
              key={skill.name}
              className="hover:scale-130 absolute flex h-10 w-10 cursor-pointer items-center justify-center rounded-full border border-border/40 bg-card/60 opacity-90 shadow-lg backdrop-blur-sm transition-all duration-300 hover:z-10 hover:border-primary/50 hover:bg-card/80 hover:opacity-100 hover:shadow-xl md:h-12 md:w-12"
              style={{
                animationDelay: `${index * ANIMATION_DELAY_STEP_SECONDS}s`,
                animationDuration: `${ANIMATION_BASE_DURATION_SECONDS + index * ANIMATION_DURATION_STEP_SECONDS}s`
              }}
            >
              <img
                src={getIconUrl(skill)}
                alt={skill.name}
                className="group-hover:scale-120 h-5 w-5 transition-transform duration-300 md:h-6 md:w-6"
              />
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
