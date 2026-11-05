import type { Project } from '../types';

/**
 * Fetches the current projects of every city from our own backend
 */
export const fetchProjects = async (): Promise<Project[]> => {
    const response = await fetch('/api/projects');
    if (!response.ok) {
        throw new Error(`Failed to load projects: server responded ${response.status}.`);
    }
    return (await response.json()) as Project[];
};
