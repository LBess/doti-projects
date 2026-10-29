import { useEffect, useState } from 'react';
import { fetchProjects } from '../lib/fetchProjects';
import type { Project, MappedProject, UnmappedProject } from '../types';

interface State {
    status: 'idle' | 'loading' | 'ready' | 'error';
    mapped: MappedProject[];
    unmapped: UnmappedProject[];
    error: string | null;
}

const INITIAL_STATE: State = { status: 'idle', mapped: [], unmapped: [], error: null };

const isMapped = (project: Project): project is MappedProject =>
    project.position != undefined && project.location != undefined;

/**
 * Loads the current DOTI projects from the backend, which has already worked
 * out where each one goes on the map. Projects it couldn't place come back in
 * `unmapped` instead.
 */
export const useProjects = (): State => {
    const [state, setState] = useState<State>(INITIAL_STATE);

    // TODO: Do this w/o an effect
    useEffect(() => {
        let cancelled = false;

        const run = async () => {
            setState({ ...INITIAL_STATE, status: 'loading' });
            try {
                const projects = await fetchProjects();
                if (projects.length === 0) {
                    throw new Error(
                        'The server has no projects yet — it may still be doing its first scrape of the DOTI page.',
                    );
                }
                if (cancelled) return;

                const mapped = projects.filter(isMapped);
                const unmapped = projects
                    .filter((p) => !isMapped(p))
                    .map((p) => ({ ...p, unmappedReason: p.unmappedReason ?? 'pending' }));
                setState({ status: 'ready', mapped, unmapped, error: null });
            } catch (err) {
                if (cancelled) return;
                setState({
                    ...INITIAL_STATE,
                    status: 'error',
                    error: err instanceof Error ? err.message : 'Failed to load DOTI projects.',
                });
            }
        };

        run();
        return () => {
            cancelled = true;
        };
    }, []);

    return state;
};
