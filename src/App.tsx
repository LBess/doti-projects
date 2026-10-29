import React from 'react';
import { APIProvider } from '@vis.gl/react-google-maps';
import { useProjects } from './hooks/useProjects';
import { ProjectMap } from './components/ProjectMap';
import { UnmappedProjects } from './components/UnmappedProjects';
import styles from './App.module.scss';

const GOOGLE_MAPS_API_KEY = import.meta.env.VITE_GOOGLE_MAPS_API_KEY as string | undefined;

const DOTI_PROJECTS_URL =
    'https://www.denvergov.org/Government/Agencies-Departments-Offices/Agencies-Departments-Offices-Directory/Department-of-Transportation-and-Infrastructure/Programs-Services/Projects';

const AppContent: React.FC = () => {
    const { status, mapped, unmapped, error } = useProjects();

    return (
        <div className={styles.app}>
            <header className={styles.appHeader}>
                <h1>Denver DOTI Current Projects</h1>
                <p>
                    Periodically refreshed from the{' '}
                    <a href={DOTI_PROJECTS_URL} target="_blank" rel="noreferrer">
                        Denver DOTI Projects page
                    </a>
                    . Pin placement is a best-effort guess based on each project's description.
                </p>
                {status === 'loading' && (
                    <p className={styles.appStatus}>Loading current projects…</p>
                )}
                {status === 'ready' && (
                    <p className={styles.appStatus}>
                        {mapped.length} projects mapped, {unmapped.length} not shown on the map.
                    </p>
                )}
                {status === 'error' && (
                    <p className={`${styles.appStatus} ${styles.appStatusError}`}>{error}</p>
                )}
            </header>

            <main className={styles.appMain}>
                <ProjectMap projects={mapped} />
                <UnmappedProjects projects={unmapped} />
            </main>
        </div>
    );
};

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
        <APIProvider apiKey={GOOGLE_MAPS_API_KEY}>
            <AppContent />
        </APIProvider>
    );
};

export default App;
