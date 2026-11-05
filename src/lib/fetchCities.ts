import type { City } from '../types';

/**
 * Fetches the cities we map, with their boundaries, from our own backend
 */
export const fetchCities = async (): Promise<City[]> => {
    const response = await fetch('/api/cities');
    if (!response.ok) {
        throw new Error(`Failed to load cities: server responded ${response.status}.`);
    }
    return (await response.json()) as City[];
};
