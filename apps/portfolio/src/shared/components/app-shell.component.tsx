import { useEffect, useRef } from 'react';
import { Outlet, useLocation } from 'react-router-dom';

import { Navigation } from './navigation.component';

export function AppShell() {
  const { pathname } = useLocation();
  const mainContent = useRef<HTMLElement>(null);

  useEffect(() => {
    mainContent.current?.scrollTo(0, 0);
  }, [pathname]);

  return (
    <>
      <div className="night-background"></div>
      <Navigation />
      <main ref={mainContent} className="flex flex-1 flex-col overflow-y-auto overflow-x-hidden">
        <Outlet />
      </main>
    </>
  );
}
