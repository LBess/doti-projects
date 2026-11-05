import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { APIProvider } from '@vis.gl/react-google-maps';
import { RouterProvider } from 'react-router/dom';
import { router } from './routes';
import styles from './App.module.scss';

const GOOGLE_MAPS_API_KEY = import.meta.env.VITE_GOOGLE_MAPS_API_KEY as string | undefined;

// The server only re-scrapes every few hours, so data is fresh for a while.
const queryClient = new QueryClient({
    defaultOptions: { queries: { staleTime: 5 * 60 * 1000 } },
});

const App: React.FC = () => {
    if (!GOOGLE_MAPS_API_KEY) {
        return (
            <div className={styles.appConfigError}>
                <h1>Missing Google Maps API key</h1>
                <p>
                    Set <code>VITE_GOOGLE_MAPS_API_KEY</code> in a <code>.env</code> file at the
                    project root (see <code>.env.example</code>), then restart the dev server.
                </p>
            </div>
        );
    }

    return (
        <QueryClientProvider client={queryClient}>
            <APIProvider apiKey={GOOGLE_MAPS_API_KEY}>
                <RouterProvider router={router} />
            </APIProvider>
        </QueryClientProvider>
    );
};

export default App;
