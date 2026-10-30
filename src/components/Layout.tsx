import { Outlet } from 'react-router';
import { useProjects } from '../hooks/useProjects';
import { Header } from './Header';
import styles from './Layout.module.scss';

export const Layout = () => {
    const projects = useProjects();

    return (
        <div className={styles.layout}>
            <Header />
            <main className={styles.main}>
                <Outlet context={projects} />
            </main>
        </div>
    );
};
