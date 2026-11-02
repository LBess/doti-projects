import clsx from 'clsx';
import { useEffect } from 'react';
import { BackIcon } from '../icons/BackIcon';
import { CloseIcon } from '../icons/CloseIcon';
import type { MappedProject } from '../types';
import styles from './ProjectDrawer.module.scss';

interface ProjectDrawerProps {
    projects: MappedProject[];
    /** The project whose details to show, or null for the list of all projects. */
    selected: MappedProject | null;
    open: boolean;
    onSelect: (id: string) => void;
    onBack: () => void;
    onClose: () => void;
}

/**
 * A panel beside the map (the map shrinks to make room; on a narrow screen it
 * slides over the map instead): the list of mapped projects, or the details of
 * the selected one.
 */
export const ProjectDrawer = ({
    projects,
    selected,
    open,
    onSelect,
    onBack,
    onClose,
}: ProjectDrawerProps) => {
    useEffect(() => {
        if (!open) return;
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') onClose();
        };
        document.addEventListener('keydown', onKeyDown);

        return () => document.removeEventListener('keydown', onKeyDown);
    }, [open, onClose]);

    return (
        // `inert` keeps the closed drawer out of the tab order and away from screen readers.
        <aside
            className={clsx(styles.drawer, open && styles.drawerOpen)}
            aria-label="Projects"
            inert={!open}
        >
            <div className={styles.drawerContent}>
                <div className={styles.drawerHeader}>
                    {selected && (
                        <button
                            type="button"
                            className={styles.iconButton}
                            aria-label="Back to all projects"
                            onClick={onBack}
                        >
                            <BackIcon />
                        </button>
                    )}
                    <h2 className={styles.drawerTitle}>
                        {selected ? 'Project details' : `Projects (${projects.length})`}
                    </h2>
                    <button
                        type="button"
                        className={styles.iconButton}
                        aria-label="Close"
                        onClick={onClose}
                    >
                        <CloseIcon />
                    </button>
                </div>

                <div className={styles.drawerBody}>
                    {selected ? (
                        <article className={styles.details}>
                            <h3>{selected.title}</h3>
                            <p className={styles.detailsLocation}>{selected.location.label}</p>
                            <p>{selected.description}</p>
                            {selected.detailUrl && (
                                <a href={selected.detailUrl} target="_blank" rel="noreferrer">
                                    Read more about {selected.title}
                                </a>
                            )}
                        </article>
                    ) : (
                        <ul className={styles.list}>
                            {projects.map((project) => (
                                <li key={project.id}>
                                    <button
                                        type="button"
                                        className={styles.listItem}
                                        onClick={() => onSelect(project.id)}
                                    >
                                        <span className={styles.listItemTitle}>
                                            {project.title}
                                        </span>
                                        <span className={styles.listItemLocation}>
                                            {project.location.label}
                                        </span>
                                    </button>
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            </div>
        </aside>
    );
};
