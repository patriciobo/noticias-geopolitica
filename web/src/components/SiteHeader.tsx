import Image from 'next/image';
import Link from 'next/link';
import NavLinks from './NavLinks';
import styles from './SiteHeader.module.css';

// Header global (layout.tsx, aparece en todas las páginas). Los links de
// nav son anclas simples a secciones que ya existen en el home — sin
// funcionalidad nueva: navegación nativa del browser (NavLinks solo marca
// la página actual).
export default function SiteHeader() {
	return (
		<header className={styles.header}>
			<div className={styles.bar}>
				<Link href='/' className={styles.brand}>
					<Image
						src='/nombre-sf.png'
						alt='Radar Global, ir al inicio'
						width={1536}
						height={1152}
						className={styles.logoName}
					/>
				</Link>
				<nav className={styles.nav} aria-label='Principal'>
					<NavLinks />
				</nav>
			</div>
			<div className={styles.accentBar} aria-hidden='true' />
		</header>
	);
}
