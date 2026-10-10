import { createBrowserRouter, Navigate } from 'react-router-dom';

import { AppShell } from '@/shared/components/app-shell.component';

// eslint-disable-next-line @typescript-eslint/naming-convention
export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppShell />,
    children: [
      { index: true, element: <Navigate to="/home" replace /> },
      {
        path: 'home',
        lazy: () =>
          import('@/pages/home/home-page.component').then(module => ({
            Component: module.HomePage
          }))
      },
      {
        path: 'experiences',
        lazy: () =>
          import('@/pages/experiences/experiences-page.component').then(module => ({
            Component: module.ExperiencesPage
          }))
      },
      {
        path: 'projects',
        lazy: () =>
          import('@/pages/projects/projects-page.component').then(module => ({
            Component: module.ProjectsPage
          }))
      }
    ]
  }
]);
