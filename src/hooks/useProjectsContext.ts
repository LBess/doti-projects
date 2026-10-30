import { useOutletContext } from 'react-router';
import type { useProjects } from './useProjects';

export type ProjectsContext = ReturnType<typeof useProjects>;

/** The projects loaded once by `Layout`, shared by every page. */
export const useProjectsContext = () => useOutletContext<ProjectsContext>();
