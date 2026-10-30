import { createBrowserRouter, Navigate } from 'react-router';
import { Layout } from './components/Layout';
import { MapPage } from './pages/MapPage';
import { AboutPage } from './pages/AboutPage';
import { UnmappedProjectsPage } from './pages/UnmappedProjectsPage';

/** What each route contributes to the header. Read from `useMatches()`. */
export interface RouteHandle {
    title: string;
    /** The index page shows the hamburger menu; every other page shows a back arrow. */
    isIndex?: boolean;
}

export class Routes {
    static UNMAPPED = 'unmapped';
    static ABOUT = 'about';
}

export const router = createBrowserRouter([
    {
        element: <Layout />,
        children: [
            {
                index: true,
                element: <MapPage />,
                handle: { title: 'Denver DOTI Projects', isIndex: true } satisfies RouteHandle,
            },
            {
                path: Routes.UNMAPPED,
                element: <UnmappedProjectsPage />,
                handle: { title: 'Unmapped Projects' } satisfies RouteHandle,
            },
            {
                path: Routes.ABOUT,
                element: <AboutPage />,
                handle: { title: 'About' } satisfies RouteHandle,
            },
            { path: '*', element: <Navigate to="/" replace /> },
        ],
    },
]);
