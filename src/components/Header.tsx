import { useEffect, useRef, useState } from 'react';
import { Link, useMatches } from 'react-router';
import { BackIcon } from '../icons/BackIcon';
import { HamburgerIcon } from '../icons/HamburgerIcon';
import { Routes, type RouteHandle } from '../routes';
import styles from './Header.module.scss';

/**
 * On the index page: a hamburger menu and the app title. On every other page:
 * an arrow back to the index and that page's title.
 */
export const Header = () => {
    const matches = useMatches();
    const handle = matches.at(-1)?.handle as RouteHandle | undefined;
    const title = handle?.title ?? '';

    return (
        <header className={styles.header}>
            {handle?.isIndex ? (
                <HamburgerMenu />
            ) : (
                <Link to="/" className={styles.iconButton} aria-label="Back to the map">
                    <BackIcon />
                </Link>
            )}
            <h1 className={styles.title}>{title}</h1>
        </header>
    );
};

const HamburgerMenu = () => {
    const [open, setOpen] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (!open) return;
        const onPointerDown = (event: PointerEvent) => {
            if (!containerRef.current?.contains(event.target as Node)) setOpen(false);
        };
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') setOpen(false);
        };
        document.addEventListener('pointerdown', onPointerDown);
        document.addEventListener('keydown', onKeyDown);
        return () => {
            document.removeEventListener('pointerdown', onPointerDown);
            document.removeEventListener('keydown', onKeyDown);
        };
    }, [open]);

    return (
        <div className={styles.menuContainer} ref={containerRef}>
            <button
                type="button"
                className={styles.iconButton}
                aria-label="Menu"
                aria-expanded={open}
                aria-controls="main-menu"
                onClick={() => setOpen((open) => !open)}
            >
                <HamburgerIcon />
            </button>
            {open && (
                <nav id="main-menu" className={styles.menu}>
                    <Link
                        to={`/${Routes.UNMAPPED}`}
                        className={styles.menuItem}
                        onClick={() => setOpen(false)}
                    >
                        Unmapped Projects
                    </Link>
                    <Link
                        to={`/${Routes.ABOUT}`}
                        className={styles.menuItem}
                        onClick={() => setOpen(false)}
                    >
                        About
                    </Link>
                </nav>
            )}
        </div>
    );
};
