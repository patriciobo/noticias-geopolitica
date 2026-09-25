import Image from 'next/image';
import Link from 'next/link';
import styles from './SiteHeader.module.css';

// Header global (layout.tsx, aparece en todas las páginas). Los links de
// nav son anclas simples a secciones que ya existen en el home — sin
// funcionalidad nueva, sin JS: navegación nativa del browser.
export default function SiteHeader() {
	return (
		<header className={styles.header}>
			<div className={styles.bar}>
				<Link href='/' className={styles.brand}>
					<Image
						src='/nombre-sf.png'
						alt='Radar Global'
						width={1536}
						height={1152}
						className={styles.logoName}
					/>
				</Link>
				<nav className={styles.nav}>
					<Link href='/#ediciones-anteriores'>Ediciones</Link>
					<Link href='/#resumen-por-region'>Regiones</Link>
					<Link href='/#fuentes-consultadas'>Fuentes</Link>
					<Link href='/metodologia'>Metodología</Link>
				</nav>
			</div>
			<div className={styles.accentBar} aria-hidden='true' />
		</header>
	);
}
