import { useQuery } from '@tanstack/react-query';
import { fetchCities } from '../lib/fetchCities';
import { fetchProjects } from '../lib/fetchProjects';
import type { City, MappedProject, Project, UnmappedProject } from '../types';

interface State {
    status: 'loading' | 'ready' | 'error';
    cities: City[];
    mapped: MappedProject[];
    unmapped: UnmappedProject[];
    error?: string;
}

// Stable references, so consumers' effects don't re-run on every render while
// data is still loading.
const NO_CITIES: City[] = [];
const NO_MAPPED: MappedProject[] = [];
const NO_UNMAPPED: UnmappedProject[] = [];

const isMapped = (project: Project): project is MappedProject =>
    project.position != undefined && project.location != undefined;

const loadProjects = async (): Promise<Project[]> => {
    const projects = await fetchProjects();
    if (projects.length === 0) {
        // Thrown (rather than returned) so the query retries: this is the state
        // the server is in until its first scrape finishes.
        throw new Error(
            'The server has no projects yet — it may still be doing its first scrape of the cities’ project pages.',
        );
    }
    return projects;
};

/** Splits projects into the ones the backend placed on the map and the rest. */
const splitProjects = (projects: Project[]) => ({
    mapped: projects.filter(isMapped),
    unmapped: projects
        .filter((project) => !isMapped(project))
        .map((project) => ({ ...project, unmappedReason: project.unmappedReason ?? 'pending' })),
});

/**
 * Loads the cities and their current projects from the backend, which has
 * already worked out where each project goes on the map. Projects it couldn't
 * place come back in `unmapped` instead.
 */
export const useProjects = (): State => {
    const citiesQuery = useQuery({
        queryKey: ['cities'],
        queryFn: fetchCities,
        // City boundaries only change when the server is redeployed.
        staleTime: Infinity,
    });
    const projectsQuery = useQuery({
        queryKey: ['projects'],
        queryFn: loadProjects,
        select: splitProjects,
    });

    const failure = citiesQuery.error ?? projectsQuery.error;
    if (failure) {
        return {
            status: 'error',
            cities: NO_CITIES,
            mapped: NO_MAPPED,
            unmapped: NO_UNMAPPED,
            error: failure instanceof Error ? failure.message : 'Failed to load projects.',
        };
    }
    if (!citiesQuery.data || !projectsQuery.data) {
        return {
            status: 'loading',
            cities: NO_CITIES,
            mapped: NO_MAPPED,
            unmapped: NO_UNMAPPED,
        };
    }
    return {
        status: 'ready',
        cities: citiesQuery.data,
        mapped: projectsQuery.data.mapped,
        unmapped: projectsQuery.data.unmapped,
    };
};
