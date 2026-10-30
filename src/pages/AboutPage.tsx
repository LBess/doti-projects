import styles from './AboutPage.module.scss';

const DOTI_PROJECTS_URL =
    'https://www.denvergov.org/Government/Agencies-Departments-Offices/Agencies-Departments-Offices-Directory/Department-of-Transportation-and-Infrastructure/Programs-Services/Projects';

export const AboutPage = () => (
    <div className={styles.page}>
        <p>
            Periodically refreshed from the{' '}
            <a href={DOTI_PROJECTS_URL} target="_blank" rel="noreferrer">
                Denver DOTI Projects page
            </a>
            . Pin placement is a best-effort guess based on each project's description.
        </p>
    </div>
);
