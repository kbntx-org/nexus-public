import { Briefcase, Github, Home, Linkedin, Rocket } from 'lucide-react';
import { NavLink } from 'react-router-dom';

import { GITHUB_URL, LINKEDIN_URL } from '@/shared/lib/social-links.constants';

const NAVIGATION_ITEMS = [
  { path: '/home', label: 'Home', icon: Home },
  { path: '/experiences', label: 'Experiences', icon: Briefcase },
  { path: '/projects', label: 'Projects', icon: Rocket }
];

const LINK_CLASS =
  'flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-all duration-300 hover:bg-white/10 hover:text-white sm:gap-1.5 sm:px-3 sm:py-1.5 sm:text-sm';

export function Navigation() {
  return (
    <>
      <nav className="fixed bottom-4 left-0 right-0 z-40 flex justify-center px-4 sm:bottom-auto sm:top-0 sm:pt-4">
        <div className="flex h-10 items-center gap-0.5 rounded-full border border-white/10 bg-white/5 px-1.5 shadow-lg backdrop-blur-md transition-all duration-300 sm:h-12 sm:gap-1 sm:px-2">
          {NAVIGATION_ITEMS.map(item => (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                `${LINK_CLASS} ${isActive ? 'bg-white/15 text-white' : 'text-white/60'}`
              }
            >
              <item.icon className="h-3.5 w-3.5 sm:h-4 sm:w-4" />
              <span>{item.label}</span>
            </NavLink>
          ))}

          <div className="mx-1.5 h-5 w-px bg-white/15 sm:hidden"></div>

          <div className="flex items-center gap-2 sm:hidden">
            <a
              href={LINKEDIN_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[#0077b5] text-white transition-all duration-300 hover:brightness-110"
            >
              <Linkedin size={14} />
            </a>
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[#333] text-white transition-all duration-300 hover:brightness-125"
            >
              <Github size={14} />
            </a>
          </div>
        </div>
      </nav>

      <div className="hidden h-16 sm:block"></div>
    </>
  );
}
