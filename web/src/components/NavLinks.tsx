"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const LINKS = [
  { href: "/#ediciones-anteriores", label: "Ediciones" },
  { href: "/#resumen-por-region", label: "Regiones" },
  { href: "/#fuentes-consultadas", label: "Fuentes" },
  { href: "/metodologia", label: "Metodología" },
];

// Client component solo para marcar la página actual (aria-current), así
// un lector de pantalla anuncia en qué sección está el usuario.
export default function NavLinks() {
  const pathname = usePathname();
  return (
    <ul>
      {LINKS.map((l) => (
        <li key={l.href}>
          <Link href={l.href} aria-current={pathname === l.href ? "page" : undefined}>
            {l.label}
          </Link>
        </li>
      ))}
    </ul>
  );
}
