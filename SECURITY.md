# Política de seguridad

## Reportar una vulnerabilidad

Si encontraste una vulnerabilidad, **no abras un issue público**. Reportala
en privado desde la pestaña **Security → Report a vulnerability** de este
repositorio (GitHub private vulnerability reporting).

Incluí, si podés:

- qué componente afecta (`core/cmd/api`, el pipeline diario, el blog en `web/`, el newsletter);
- pasos para reproducirla;
- el impacto que ves (qué podría hacer alguien que la explote).

Vas a recibir una respuesta dentro de los 7 días. Una vez corregida, si
querés, te damos crédito en las notas del cambio.

## Alcance

Nos interesan especialmente:

- cualquier forma de alterar el contenido de un informe publicado sin que
  quede registrado (ver "Integridad de los informes" en el README);
- inyección de instrucciones al modelo a través de los titulares de un medio
  (prompt injection) que logre cambiar lo que dice el informe;
- abuso del alta al newsletter (suscribir direcciones ajenas, saltear el
  doble opt-in, agotar la cuota de envío);
- exposición de datos de suscriptores o de secretos del proyecto.

Fuera de alcance: ataques de denegación de servicio volumétricos contra los
planes gratuitos de hosting, y reportes automáticos de escáneres sin una
prueba de impacto concreta.
