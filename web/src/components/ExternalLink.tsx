import type { ReactNode } from "react";

export const NEW_TAB_TEXT = "se abre en una pestaña nueva";

/**
 * Link que abre en otra pestaña. Lo avisa a los lectores de pantalla: un
 * cambio de contexto sin anuncio desorienta a quien no ve la pantalla.
 */
export default function ExternalLink({
  href,
  className,
  children,
}: {
  href?: string;
  className?: string;
  children?: ReactNode;
}) {
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className={className}>
      {children}
      <span className="sr-only"> ({NEW_TAB_TEXT})</span>
    </a>
  );
}
